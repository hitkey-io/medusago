// Package composer mirrors @medusajs/workflows-sdk/src/utils/composer:
// createWorkflow, createStep, createHook and StepResponse.
//
// Upstream composes a workflow into a static step tree (through proxies)
// that the TransactionOrchestrator executes. In Go a workflow body is plain
// code: steps run when they are called, transform() and when() are ordinary
// Go, parallelize() runs its branches in order. The observable semantics of
// a synchronous run are the same:
//
//   - steps run in order; a step error is retried maxRetries times, then
//     fails the transaction (unless continueOnPermanentFailure);
//   - on failure every step that completed, and the failed step itself, is
//     compensated in reverse order; a step that failed receives undefined
//     (nil) as compensation input;
//   - run() returns the first invoke error, unchanged;
//   - events emitted with the workflow's event group are released when the
//     workflow succeeds and cleared when it fails;
//   - nested workflows (runAsStep) compensate as a unit.
//
// Synchronous executions upstream leave no workflow_execution row behind (the
// row written at start is deleted on finish), so nothing is persisted here.
// Async steps, retention and idempotent runs are not supported yet.
package composer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"github.com/oklog/ulid/v2"

	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// Container resolves registrations (the workflow container / req.scope).
type Container interface {
	Resolve(key string) any
}

// StepExecutionContext mirrors StepExecutionContext.
type StepExecutionContext struct {
	Context       context.Context
	Container     Container
	WorkflowID    string
	StepName      string
	TransactionID string
	// Action is "invoke" or "compensate".
	Action       string
	EventGroupID string
	// Attempt is the 1-based invoke attempt.
	Attempt int
	exec    *Execution
}

// Execution returns the running execution (for nested workflows).
func (c StepExecutionContext) Execution() *Execution { return c.exec }

// StepResponse mirrors StepResponse: the step output and the input given to
// its compensation (the output itself when not set).
type StepResponse[O any] struct {
	Output          O
	CompensateInput any
	hasCompensate   bool
	permanentFail   error
}

// NewStepResponse mirrors new StepResponse(output, compensateInput?).
func NewStepResponse[O any](output O, compensateInput ...any) StepResponse[O] {
	r := StepResponse[O]{Output: output}
	if len(compensateInput) > 0 {
		r.CompensateInput = compensateInput[0]
		r.hasCompensate = true
	}
	return r
}

// PermanentFailure mirrors StepResponse.permanentFailure(message).
func PermanentFailure[O any](message string) StepResponse[O] {
	return StepResponse[O]{permanentFail: errors.New(message)}
}

// StepOptions mirrors the createStep / .config() options supported here.
type StepOptions struct {
	MaxRetries                 int
	ContinueOnPermanentFailure bool
	NoCompensation             bool
}

// Step is a step definition.
type Step[I, O any] struct {
	Name       string
	Options    StepOptions
	invoke     func(StepExecutionContext, I) (StepResponse[O], error)
	compensate func(StepExecutionContext, any) error
}

// CreateStep mirrors createStep(name, invoke, compensate?).
func CreateStep[I, O any](name string, invoke func(StepExecutionContext, I) (StepResponse[O], error), compensate func(StepExecutionContext, any) error) *Step[I, O] {
	return &Step[I, O]{Name: name, invoke: invoke, compensate: compensate}
}

// Config mirrors step(...).config({ name, ...options }): a copy with a new
// name (empty keeps it) and options.
func (s *Step[I, O]) Config(name string, opts StepOptions) *Step[I, O] {
	c := *s
	if name != "" {
		c.Name = name
	}
	c.Options = opts
	return &c
}

type compensation struct {
	name string
	run  func() error
}

// Execution is one workflow run (a DistributedTransaction).
type Execution struct {
	ctx           context.Context
	container     Container
	workflowID    string
	transactionID string
	eventGroupID  string
	compensations []compensation
	mu            sync.Mutex
	// errors mirrors transaction.getErrors(INVOKE).
	errors []error
	hooks  *hookRegistry
}

// Context returns the run context.
func (e *Execution) Context() context.Context { return e.ctx }

// Container returns the run container.
func (e *Execution) Container() Container { return e.container }

// WorkflowID returns the workflow id.
func (e *Execution) WorkflowID() string { return e.workflowID }

// TransactionID returns the transaction id.
func (e *Execution) TransactionID() string { return e.transactionID }

// EventGroupID returns the event group of the run.
func (e *Execution) EventGroupID() string { return e.eventGroupID }

func (e *Execution) pushCompensation(c compensation) {
	e.mu.Lock()
	e.compensations = append(e.compensations, c)
	e.mu.Unlock()
}

func (e *Execution) stepContext(name, action string, attempt int) StepExecutionContext {
	return StepExecutionContext{
		Context: e.ctx, Container: e.container, WorkflowID: e.workflowID, StepName: name,
		TransactionID: e.transactionID, Action: action, EventGroupID: e.eventGroupID, Attempt: attempt, exec: e,
	}
}

// compensate runs the compensations in reverse order. Compensation errors
// are logged; they do not replace the invoke error (upstream run() throws
// the first INVOKE error).
func (e *Execution) compensate() {
	for i := len(e.compensations) - 1; i >= 0; i-- {
		c := e.compensations[i]
		if err := c.run(); err != nil {
			slog.Error(fmt.Sprintf("%s:%s:compensate - %v", e.workflowID, c.name, err))
		}
	}
	e.compensations = nil
}

// callSafely runs fn and converts a panic into an error, like a thrown
// exception upstream.
func callSafely(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("%v", r)
			}
			slog.Error(fmt.Sprintf("panic in workflow step: %v\n%s", r, debug.Stack()))
		}
	}()
	return fn()
}

// stepFailed is returned to abort the workflow body after a step failed
// permanently. It wraps the original error.
type stepFailed struct{ err error }

func (s *stepFailed) Error() string { return s.err.Error() }
func (s *stepFailed) Unwrap() error { return s.err }

// Run invokes the step inside the workflow execution.
func (s *Step[I, O]) Run(e *Execution, input I) (O, error) {
	var zero O
	var res StepResponse[O]
	var err error
	attempts := s.Options.MaxRetries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		ctx := e.stepContext(s.Name, "invoke", attempt)
		err = callSafely(func() error {
			var ierr error
			res, ierr = s.invoke(ctx, input)
			return ierr
		})
		if err == nil && res.permanentFail != nil {
			err = res.permanentFail
			break
		}
		if err == nil {
			break
		}
		var sf *stepFailed
		if errors.As(err, &sf) {
			// a nested workflow already failed permanently
			break
		}
	}
	if err != nil {
		e.mu.Lock()
		e.errors = append(e.errors, err)
		e.mu.Unlock()
		if s.Options.ContinueOnPermanentFailure {
			return zero, nil
		}
		// the failed step is compensated too, with undefined input
		if s.compensate != nil && !s.Options.NoCompensation {
			comp := s.compensate
			name := s.Name
			e.pushCompensation(compensation{name: name, run: func() error {
				return callSafely(func() error { return comp(e.stepContext(name, "compensate", 1), nil) })
			}})
		}
		return zero, &stepFailed{err: err}
	}
	if s.compensate != nil && !s.Options.NoCompensation {
		var compInput any = res.Output
		if res.hasCompensate {
			compInput = res.CompensateInput
		}
		comp := s.compensate
		name := s.Name
		e.pushCompensation(compensation{name: name, run: func() error {
			return callSafely(func() error { return comp(e.stepContext(name, "compensate", 1), compInput) })
		}})
	}
	return res.Output, nil
}

// Workflow is a workflow definition.
type Workflow[I, O any] struct {
	ID    string
	body  func(*Execution, I) (O, error)
	hooks *hookRegistry
}

// CreateWorkflow mirrors createWorkflow(id, composer).
func CreateWorkflow[I, O any](id string, body func(*Execution, I) (O, error)) *Workflow[I, O] {
	return &Workflow[I, O]{ID: id, body: body, hooks: newHookRegistry()}
}

// RunOptions mirrors the options of workflow(container).run().
type RunOptions struct {
	// TransactionID defaults to "auto-<ulid>".
	TransactionID string
	// EventGroupID defaults to a new ulid.
	EventGroupID string
}

// Run mirrors workflow(container).run({ input }): the result, or the first
// invoke error (throwOnError true).
func (w *Workflow[I, O]) Run(ctx context.Context, c Container, input I, opts ...RunOptions) (O, error) {
	var o RunOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	if o.TransactionID == "" {
		o.TransactionID = "auto-" + ulid.Make().String()
	}
	if o.EventGroupID == "" {
		o.EventGroupID = ulid.Make().String()
	}
	e := &Execution{ctx: ctx, container: c, workflowID: w.ID, transactionID: o.TransactionID, eventGroupID: o.EventGroupID, hooks: w.hooks}
	out, err := w.execute(e, input)
	bus, _ := c.Resolve(utilsmodulessdk.EVENT_BUS).(EventBus)
	if err != nil {
		if bus != nil {
			_ = bus.ClearGroupedEvents(ctx, e.eventGroupID, nil)
		}
		var zero O
		return zero, err
	}
	if bus != nil {
		if rerr := bus.ReleaseGroupedEvents(ctx, e.eventGroupID); rerr != nil {
			slog.Error(fmt.Sprintf("Failed to release grouped events for eventGroupId: %s: %v", e.eventGroupID, rerr))
		}
	}
	return out, nil
}

// execute runs the body; on failure it compensates and returns the first
// invoke error.
func (w *Workflow[I, O]) execute(e *Execution, input I) (O, error) {
	var out O
	err := callSafely(func() error {
		var berr error
		out, berr = w.body(e, input)
		return berr
	})
	if err != nil {
		e.compensate()
		var zero O
		if len(e.errors) > 0 {
			return zero, e.errors[0]
		}
		var sf *stepFailed
		if errors.As(err, &sf) {
			return zero, sf.err
		}
		return zero, err
	}
	return out, nil
}

// RunAsStep mirrors workflow.runAsStep({ input }): the nested workflow runs
// within the parent execution (same container, event group) and is
// compensated as a unit when the parent fails.
func (w *Workflow[I, O]) RunAsStep(parent *Execution, input I) (O, error) {
	e := &Execution{ctx: parent.ctx, container: parent.container, workflowID: w.ID, transactionID: parent.transactionID, eventGroupID: parent.eventGroupID, hooks: w.hooks}
	out, err := w.execute(e, input)
	if err != nil {
		parent.mu.Lock()
		parent.errors = append(parent.errors, err)
		parent.mu.Unlock()
		var zero O
		return zero, &stepFailed{err: err}
	}
	nested := e.compensations
	parent.pushCompensation(compensation{name: w.ID + "-as-step", run: func() error {
		e.compensations = nested
		e.compensate()
		return nil
	}})
	return out, nil
}

// Parallelize mirrors parallelize(...): the branches run in order (upstream
// they run concurrently; their relative order is not observable). Like the
// orchestrator, every branch runs even when an earlier one fails, so the
// steps that succeeded are compensated with the failed one; the first
// error is returned.
func Parallelize(fns ...func() error) error {
	var first error
	for _, fn := range fns {
		if err := fn(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
