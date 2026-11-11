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
// Every run keeps a journal of its steps in the TransactionFlow /
// TransactionContext format (see journal.go). It is stored in
// workflow_execution like InMemoryDistributedTransactionStorage does: for
// workflows created with store / retentionTime and for runs with async
// steps; finished runs without retentionTime are deleted. A step with
// async + backgroundExecution detaches the run: run() returns while the
// body continues in the background; an async step without background
// execution waits for setStepSuccess / setStepFailure. Waiting runs live in
// this process (like the in-memory engine's timers).
package composer

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

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
	// Async marks a step completed from outside (setStepSuccess) unless
	// BackgroundExecution is set, in which case its handler runs in the
	// background and the run returns to the caller.
	Async               bool
	BackgroundExecution bool
	Store               bool
	// Timeout (seconds) fails an async step still waiting for a response.
	Timeout int
}

// Step is a step definition.
type Step[I, O any] struct {
	Name       string
	Options    StepOptions
	invoke     func(StepExecutionContext, I) (StepResponse[O], error)
	compensate func(StepExecutionContext, any) error
	// action is the flow action name when it differs from Name (hooks).
	action string
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

// detacher lets the first background / async step return the run to its
// caller.
type detacher struct {
	once sync.Once
	ch   chan struct{}
}

func newDetacher() *detacher { return &detacher{ch: make(chan struct{})} }

func (d *detacher) detach() { d.once.Do(func() { close(d.ch) }) }

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
	// shared by the root run and its runAsStep workflows
	journal  *journal
	detacher *detacher
	// partial is the result returned when the run detaches (the
	// WorkflowResponse resolved from the steps that ran).
	partial        any
	rootWorkflowID string
	rootOptions    WorkflowOptions
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
	action := s.Name
	if s.action != "" {
		action = s.action
	}
	noComp := s.compensate == nil || s.Options.NoCompensation
	rec := e.journal.begin(action, s.Options, noComp)
	background := s.Options.Async && s.Options.BackgroundExecution
	if background {
		// the run returns to its caller; the step runs in the background
		e.persist(TransactionStateInvoking)
		e.detacher.detach()
	}
	attempts := s.Options.MaxRetries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		e.journal.attempt(rec, attempt)
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
	// an async step whose handler returned nothing waits for
	// setStepSuccess / setStepFailure; one that returned a response
	// completes with it (upstream registers the returned response)
	if err == nil && s.Options.Async && !background && isNilOutput(any(res.Output)) {
		// waits for setStepSuccess / setStepFailure; the run returns to its
		// caller meanwhile
		coord := stepCoordinator(e.container)
		key := IdempotencyKey{Action: "invoke", TransactionID: e.transactionID, StepID: action, WorkflowID: e.rootWorkflowID}
		ch := coord.Wait(key)
		e.journal.wait(rec)
		e.persist(TransactionStateInvoking)
		e.detacher.detach()
		var r StepResult
		if s.Options.Timeout > 0 {
			select {
			case r = <-ch:
			case <-time.After(time.Duration(s.Options.Timeout) * time.Second):
				coord.Cancel(key)
				r = StepResult{Err: &TransactionStepTimeoutError{}}
			}
		} else {
			r = <-ch
		}
		if r.Err != nil {
			err = r.Err
		} else {
			out, cerr := convertOutput[O](r.Output)
			if cerr != nil {
				err = cerr
			} else {
				res = StepResponse[O]{Output: out}
			}
		}
	}
	if err != nil {
		e.journal.fail(rec, err)
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
				cerr := callSafely(func() error { return comp(e.stepContext(name, "compensate", 1), nil) })
				e.journal.compensated(rec, cerr)
				return cerr
			}})
		}
		return zero, &stepFailed{err: err}
	}
	var compInput any = res.Output
	if res.hasCompensate {
		compInput = res.CompensateInput
	}
	e.journal.succeed(rec, res.Output, any(res.Output) != nil, compInput, res.hasCompensate)
	if s.compensate != nil && !s.Options.NoCompensation {
		comp := s.compensate
		name := s.Name
		e.pushCompensation(compensation{name: name, run: func() error {
			cerr := callSafely(func() error { return comp(e.stepContext(name, "compensate", 1), compInput) })
			e.journal.compensated(rec, cerr)
			return cerr
		}})
	}
	return res.Output, nil
}

// Workflow is a workflow definition.
type Workflow[I, O any] struct {
	ID      string
	Options WorkflowOptions
	body    func(*Execution, I) (O, error)
	hooks   *hookRegistry
}

// CreateWorkflow mirrors createWorkflow(id, composer).
func CreateWorkflow[I, O any](id string, body func(*Execution, I) (O, error)) *Workflow[I, O] {
	registerWorkflow(id)
	w := &Workflow[I, O]{ID: id, body: body, hooks: newHookRegistry()}
	registerRunner(w)
	return w
}

// CreateWorkflowWithOptions mirrors createWorkflow({ name, store,
// retentionTime, idempotent }, composer).
func CreateWorkflowWithOptions[I, O any](id string, opts WorkflowOptions, body func(*Execution, I) (O, error)) *Workflow[I, O] {
	registerWorkflow(id)
	w := &Workflow[I, O]{ID: id, Options: opts, body: body, hooks: newHookRegistry()}
	registerRunner(w)
	if opts.Schedule != nil {
		registerScheduled(id, *opts.Schedule)
	}
	return w
}

// RunOptions mirrors the options of workflow(container).run().
type RunOptions struct {
	// TransactionID defaults to "auto-<ulid>".
	TransactionID string
	// EventGroupID defaults to a new ulid.
	EventGroupID string
	// Notify delivers the run's events to the workflow engine subscribers
	// (runs started through the workflow engine).
	Notify bool
}

// Run mirrors workflow(container).run({ input }): the result, or the first
// invoke error (throwOnError true).
func (w *Workflow[I, O]) Run(ctx context.Context, c Container, input I, opts ...RunOptions) (O, error) {
	out, _, err := w.RunWithInfo(ctx, c, input, opts...)
	return out, err
}

// RunWithInfo is Run that also describes the transaction (the
// { transaction } of workflow.run()).
func (w *Workflow[I, O]) RunWithInfo(ctx context.Context, c Container, input I, opts ...RunOptions) (O, RunInfo, error) {
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
	// runs are not tied to the caller (a request may end before a
	// background step does)
	runCtx := context.WithoutCancel(ctx)
	e := &Execution{ctx: runCtx, container: c, workflowID: w.ID, transactionID: o.TransactionID, eventGroupID: o.EventGroupID, hooks: w.hooks,
		journal: newJournal(input), detacher: newDetacher(), rootWorkflowID: w.ID, rootOptions: w.Options}
	if o.Notify {
		workflowID, transactionID := w.ID, o.TransactionID
		publisher := eventPublisher(c)
		e.journal.events = func(ev WorkflowEvent) {
			ev.WorkflowID, ev.TransactionID = workflowID, transactionID
			// called with the journal locked
			ev.IsFlowAsync = e.journal.hasAsync
			if ev.EventType == "onFinish" {
				ev.State = e.journal.state
			}
			notifyEvent(ev)
			if publisher != nil {
				publisher.PublishWorkflowEvent(ev)
			}
		}
		e.journal.emitRun("onBegin", nil)
	}
	stored := w.Options.Store || w.Options.RetentionTime > 0
	if stored {
		e.persist(TransactionStateNotStarted)
	}
	type result struct {
		out O
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := w.execute(e, input)
		done <- result{out, err}
	}()
	select {
	case r := <-done:
		state := w.finish(e, r.err, stored)
		e.emitFinish(r.out, r.err)
		info := RunInfo{TransactionID: e.transactionID, RunID: e.journal.runID, State: state}
		if r.err != nil {
			var zero O
			return zero, info, r.err
		}
		return r.out, info, nil
	case <-e.detacher.ch:
		go func() {
			r := <-done
			w.finish(e, r.err, true)
			e.emitFinish(r.out, r.err)
		}()
		var zero O
		if p, ok := e.partial.(O); ok {
			zero = p
		}
		return zero, RunInfo{TransactionID: e.transactionID, RunID: e.journal.runID, State: TransactionStateInvoking, Detached: true}, nil
	}
}

// finish releases or clears the grouped events and stores or deletes the
// checkpoint of a finished run; it returns the final state.
func (w *Workflow[I, O]) finish(e *Execution, err error, stored bool) string {
	bus, _ := e.container.Resolve(utilsmodulessdk.EVENT_BUS).(EventBus)
	state := TransactionStateDone
	if err != nil {
		state = TransactionStateReverted
		if bus != nil {
			_ = bus.ClearGroupedEvents(e.ctx, e.eventGroupID, nil)
		}
	} else if bus != nil {
		if rerr := bus.ReleaseGroupedEvents(e.ctx, e.eventGroupID); rerr != nil {
			slog.Error(fmt.Sprintf("Failed to release grouped events for eventGroupId: %s: %v", e.eventGroupID, rerr))
		}
	}
	if err != nil {
		e.journal.revertAll()
		if e.journal.hasFailed {
			state = TransactionStateFailed
		}
	}
	e.journal.setState(state)
	if stored || e.journal.hasAsync {
		if w.Options.RetentionTime > 0 {
			e.persist(state)
		} else {
			e.removeExecution()
		}
	}
	return state
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
		e.journal.emitRun("onCompensateBegin", nil)
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
	e := &Execution{ctx: parent.ctx, container: parent.container, workflowID: w.ID, transactionID: parent.transactionID, eventGroupID: parent.eventGroupID, hooks: w.hooks,
		journal: parent.journal, detacher: parent.detacher, rootWorkflowID: parent.rootWorkflowID, rootOptions: parent.rootOptions}
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

// SetPartialResult sets what run() returns when the run detaches before the
// body finishes (the WorkflowResponse value already resolved by the steps
// that ran before an async step).
func SetPartialResult(e *Execution, v any) {
	e.partial = v
}

// TransactionStepTimeoutError mirrors TransactionStepTimeoutError.
type TransactionStepTimeoutError struct{}

func (*TransactionStepTimeoutError) Error() string { return "Timeout error" }

func isNilOutput(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		return rv.IsNil()
	}
	return false
}

// emitFinish mirrors the onFinish notification of the orchestrator's run():
// the result (undefined for a failed run) and the transaction errors.
func (e *Execution) emitFinish(out any, err error) {
	e.journal.emitRun("onFinish", func(ev *WorkflowEvent) {
		if err == nil {
			ev.Result, ev.HasResult = out, true
		}
		ev.Errors, ev.HasErrors = append([]any{}, e.journal.errors...), true
	})
}
