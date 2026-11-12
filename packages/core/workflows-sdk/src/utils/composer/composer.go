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
// execution waits for setStepSuccess / setStepFailure. With the in-memory
// engine waiting runs live in this process (like its timers); a durable
// engine (Redis) stops the run at the waiting step and resumes it from its
// checkpoint on any instance (see resume.go).
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

	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
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
	// compResponse is set by a compensation that returns a StepResponse
	compResponse *bool
}

// ReturnStepResponse mirrors a compensation returning new StepResponse():
// the context records its response ({ output: StepResponse }) instead of
// undefined.
func (c StepExecutionContext) ReturnStepResponse() {
	if c.compResponse != nil {
		*c.compResponse = true
	}
}

// Execution returns the running execution (for nested workflows).
func (c StepExecutionContext) Execution() *Execution { return c.exec }

// IsSubWorkflow reports whether the step runs in a workflow executed as a
// step of another one (parentStepIdempotencyKey is set upstream).
func (c StepExecutionContext) IsSubWorkflow() bool {
	return c.exec != nil && c.exec.subWorkflow
}

// StepResponse mirrors StepResponse: the step output and the input given to
// its compensation (the output itself when not set).
type StepResponse[O any] struct {
	Output          O
	CompensateInput any
	hasCompensate   bool
	permanentFail   error
	skip            bool
	// nullOutput: the output is null (new StepResponse(null)), not
	// undefined
	nullOutput bool
}

// SkipStep mirrors StepResponse.skip(): the step is marked skipped, its
// output is undefined and it is not compensated.
func SkipStep[O any]() StepResponse[O] {
	return StepResponse[O]{skip: true}
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

// NewNullStepResponse mirrors new StepResponse(null, compensateInput?): the
// output is stored as null (NewStepResponse with a nil output leaves it
// undefined).
func NewNullStepResponse[O any](compensateInput ...any) StepResponse[O] {
	r := NewStepResponse[O](*new(O), compensateInput...)
	r.nullOutput = true
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
	sharedAggregator     *eventbus.SharedAggregator
	sharedAggregatorOnce sync.Once
	ctx                  context.Context
	container            Container
	workflowID           string
	transactionID        string
	eventGroupID         string
	compensations        []compensation
	mu                   sync.Mutex
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
	// subWorkflow: the workflow runs as a step of another one (parentExec)
	subWorkflow bool
	parentExec  *Execution
	// replay is set when the run resumes a stored transaction
	replay *replayState
	// persistedRev is the journal revision of the last persist
	persistedRev int
	// cancelled: cancelTransaction was called on the (root) run
	cancelled bool
	// finalState is the state of a detached run that finished
	finalState string
	// finished is closed once the (root) run finished
	finished     chan struct{}
	finishedOnce sync.Once
}

// Context returns the run context.
func (e *Execution) Context() context.Context { return e.ctx }

// Container returns the run container.
func (e *Execution) Container() Container { return e.container }

// WorkflowID returns the workflow id.
func (e *Execution) WorkflowID() string { return e.workflowID }

// TransactionID returns the transaction id.
func (e *Execution) TransactionID() string { return e.transactionID }

// aggregator returns the message aggregator of the run's medusaContext.
func (e *Execution) aggregator() *eventbus.SharedAggregator {
	e.sharedAggregatorOnce.Do(func() { e.sharedAggregator = eventbus.NewSharedAggregator() })
	return e.sharedAggregator
}

// EventGroupID returns the event group of the run.
func (e *Execution) EventGroupID() string { return e.eventGroupID }

func (e *Execution) pushCompensation(c compensation) {
	e.mu.Lock()
	e.compensations = append(e.compensations, c)
	e.mu.Unlock()
}

func (e *Execution) stepContext(name, action string, attempt int) StepExecutionContext {
	return StepExecutionContext{
		// the module services called from a step get the run's event group
		// and message aggregator (the workflow medusaContext)
		Context: eventbus.WithSharedAggregator(eventbus.WithEventGroupID(e.ctx, e.eventGroupID), e.aggregator()), Container: e.container, WorkflowID: e.workflowID, StepName: name,
		TransactionID: e.transactionID, Action: action, EventGroupID: e.eventGroupID, Attempt: attempt, exec: e,
	}
}

// compensateContext is the step context of a compensation and the flag it
// sets when it returns a StepResponse.
func (e *Execution) compensateContext(name string) (StepExecutionContext, *bool) {
	ctx := e.stepContext(name, "compensate", 1)
	resp := new(bool)
	ctx.compResponse = resp
	return ctx, resp
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
	action := s.Name
	if s.action != "" {
		action = s.action
	}
	noComp := s.compensate == nil || s.Options.NoCompensation
	if e.replay != nil {
		if out, handled, rerr := s.replayStep(e, action, noComp, input); handled {
			return out, rerr
		}
	}
	// a cancelled transaction runs no further step (cancelTransaction)
	if e.isCancelled() {
		return zero, errRunCancelled
	}
	rec := e.journal.begin(action, s.Options, noComp)
	background := s.Options.Async && s.Options.BackgroundExecution
	if background {
		// the run returns to its caller; the step runs in the background
		e.persist(TransactionStateInvoking)
		e.detacher.detach()
	}
	return s.invokeFrom(e, rec, action, input, 1, background)
}

// invokeFrom invokes the step handler from the given attempt (retries
// included), waits for the response of an async step and settles the
// step. retryStep re-invokes a waiting step through it.
func (s *Step[I, O]) invokeFrom(e *Execution, rec *stepRecord, action string, input I, first int, background bool) (O, error) {
	var zero O
	var res StepResponse[O]
	var err error
	invoke := func(attempt int) {
		e.journal.attempt(rec, attempt)
		ctx := e.stepContext(s.Name, "invoke", attempt)
		err = callSafely(func() error {
			var ierr error
			res, ierr = s.invoke(ctx, input)
			return ierr
		})
		if err == nil && res.permanentFail != nil {
			err = res.permanentFail
		}
	}
	last := first + s.Options.MaxRetries
	for attempt := first; attempt <= last; attempt++ {
		invoke(attempt)
		if err == nil {
			break
		}
		var sf *stepFailed
		if errors.As(err, &sf) || res.permanentFail != nil {
			// a nested workflow already failed permanently, or the step
			// failed permanently: no retry
			break
		}
	}
	// an async step whose handler returned nothing waits for
	// setStepSuccess / setStepFailure; one that returned a response
	// completes with it (upstream registers the returned response)
	if err == nil && s.Options.Async && !background && isNilOutput(any(res.Output)) {
		key := IdempotencyKey{Action: "invoke", TransactionID: e.transactionID, StepID: action, WorkflowID: e.rootWorkflowID}
		if de := durableEngine(e.container); de != nil {
			// the checkpoint holds the waiting step: the response resumes
			// the transaction on any instance; this run stops here
			e.journal.wait(rec)
			e.persist(TransactionStateInvoking)
			if s.Options.Timeout > 0 {
				if terr := de.ScheduleStepTimeout(context.WithoutCancel(e.ctx), e.stepTimeoutJob(rec)); terr != nil {
					slog.Error(fmt.Sprintf("failed to schedule the timeout of step %s: %v", rec.id, terr))
				}
			}
			e.detacher.detach()
			return zero, errSuspended
		}
		// waits for setStepSuccess / setStepFailure (or retryStep); the run
		// returns to its caller meanwhile
		coord := stepCoordinator(e.container)
		var deadline <-chan time.Time
		if s.Options.Timeout > 0 {
			deadline = time.After(time.Duration(s.Options.Timeout) * time.Second)
		}
		for {
			ch := coord.Wait(key)
			e.journal.wait(rec)
			e.persist(TransactionStateInvoking)
			e.detacher.detach()
			var r StepResult
			select {
			case r = <-ch:
			case <-deadline:
				coord.Forget(key)
				r = StepResult{Err: &TransactionStepTimeoutError{}}
			}
			if r.Retry {
				// retryStep: the handler runs again; an empty response
				// waits again
				invoke(rec.attempts + 1)
				if err == nil && isNilOutput(any(res.Output)) {
					continue
				}
				break
			}
			res, err = applyStepResult[O](r)
			if r.Err == nil {
				rec.raw, rec.hasRaw = r.Output, true
			}
			break
		}
	}
	out, err := s.settle(e, rec, res, err)
	if err == nil && e.isCancelled() {
		// the transaction was cancelled while the step ran or waited: it
		// is compensated now (waiting_to_compensate -> compensating)
		return zero, errRunCancelled
	}
	return out, err
}

// applyStepResult converts the response delivered to a waiting step.
func applyStepResult[O any](r StepResult) (StepResponse[O], error) {
	if r.Err != nil {
		return StepResponse[O]{}, r.Err
	}
	out, err := convertOutput[O](r.Output)
	if err != nil {
		return StepResponse[O]{}, err
	}
	return StepResponse[O]{Output: out}, nil
}

// settle records the outcome of an invoked step and registers its
// compensation.
func (s *Step[I, O]) settle(e *Execution, rec *stepRecord, res StepResponse[O], err error) (O, error) {
	var zero O
	if err != nil {
		e.journal.fail(rec, err)
		e.mu.Lock()
		e.errors = append(e.errors, err)
		e.mu.Unlock()
		e.afterDelivered(rec)
		if s.Options.ContinueOnPermanentFailure {
			return zero, nil
		}
		// the failed step is compensated too, with undefined input
		if s.compensate != nil && !s.Options.NoCompensation {
			comp := s.compensate
			name := s.Name
			e.pushCompensation(compensation{name: name, run: func() error {
				e.journal.compensateBegin(rec)
				ctx, resp := e.compensateContext(name)
				cerr := callSafely(func() error { return comp(ctx, nil) })
				e.journal.compensated(rec, cerr, *resp)
				return cerr
			}})
		}
		return zero, &stepFailed{err: err}
	}
	if res.skip {
		e.journal.skip(rec, true)
		e.afterDelivered(rec)
		return zero, nil
	}
	var compInput any = res.Output
	if res.hasCompensate {
		compInput = res.CompensateInput
	}
	e.journal.succeed(rec, res.Output, any(res.Output) != nil || res.nullOutput, compInput, res.hasCompensate)
	e.afterDelivered(rec)
	s.pushStepCompensation(e, rec, compInput)
	return res.Output, nil
}

func (s *Step[I, O]) pushStepCompensation(e *Execution, rec *stepRecord, compInput any) {
	if s.compensate == nil || s.Options.NoCompensation {
		return
	}
	comp := s.compensate
	name := s.Name
	e.pushCompensation(compensation{name: name, run: func() error {
		e.journal.compensateBegin(rec)
		ctx, resp := e.compensateContext(name)
		cerr := callSafely(func() error { return comp(ctx, compInput) })
		e.journal.compensated(rec, cerr, *resp)
		return cerr
	}})
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
	e.journal.define(w.ID)
	e.journal.engineRun = o.Notify
	e.registerLive()
	if o.Notify {
		e.notifyEvents()
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
		if errors.Is(r.err, errSuspended) {
			// the run waits for a response; the checkpoint resumes it
			e.forgetLive()
			e.persistIfChanged(TransactionStateInvoking)
			var zero O
			if p, ok := e.partial.(O); ok {
				zero = p
			}
			return zero, RunInfo{TransactionID: e.transactionID, RunID: e.journal.runID, State: TransactionStateInvoking, Detached: true}, nil
		}
		state := w.finish(e, r.err, stored)
		e.emitFinish(r.out, r.err)
		e.markFinished()
		info := RunInfo{TransactionID: e.transactionID, RunID: e.journal.runID, State: state}
		if r.err != nil {
			var zero O
			return zero, info, r.err
		}
		return r.out, info, nil
	case <-e.detacher.ch:
		go func() {
			r := <-done
			if errors.Is(r.err, errSuspended) {
				// the steps run after the suspension (parallel branches)
				// are part of the checkpoint
				e.forgetLive()
				e.persistIfChanged(TransactionStateInvoking)
				return
			}
			e.finalState = w.finish(e, r.err, true)
			e.emitFinish(r.out, r.err)
			e.markFinished()
		}()
		var zero O
		if p, ok := e.partial.(O); ok {
			zero = p
		}
		return zero, RunInfo{TransactionID: e.transactionID, RunID: e.journal.runID, State: TransactionStateInvoking, Detached: true}, nil
	}
}

// notifyEvents delivers the run's notifications to the workflow engine
// subscribers (and to the other instances through the engine).
func (e *Execution) notifyEvents() {
	workflowID, transactionID := e.rootWorkflowID, e.transactionID
	publisher := eventPublisher(e.container)
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
	} else {
		e.journal.skipUnreached()
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
	if errors.Is(err, errSuspended) {
		var zero O
		return zero, errSuspended
	}
	if err == nil && e.parentExec == nil && e.isCancelled() {
		// a cancelled transaction whose steps all finished is compensated
		err = errRunCancelled
	}
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
// within the parent execution (same container, event group) as one step of
// the parent, "<id>-as-step". The nested run has its own transaction
// ("<id>-as-step-<parent transaction id>"): its steps are not steps of the
// parent; the parent step's output is the nested result and its
// compensation input the nested transaction (flow and context), like the
// WorkflowResult upstream stores. The nested run is compensated as a unit
// when the parent fails.
func (w *Workflow[I, O]) RunAsStep(parent *Execution, input I) (O, error) {
	return w.RunAsStepNamed(parent, "", input)
}

// RunAsStepNamed mirrors workflow.runAsStep({ input }).config({ name }): the
// parent step (and the nested transaction id) is named after the configured
// name; "" keeps "<id>-as-step".
func (w *Workflow[I, O]) RunAsStepNamed(parent *Execution, name string, input I) (O, error) {
	var zero O
	if name == "" {
		name = w.ID + "-as-step"
	}
	async := false
	if n := findFlowNode(flowDefinition(parent.workflowID), name); n != nil {
		async = fieldBool(n.fields, "async")
	}
	if parent.replay != nil {
		if out, handled, err := w.replayAsStep(parent, name, async, input); handled {
			return out, err
		}
	}
	rec := parent.journal.begin(name, StepOptions{Async: async}, false)
	e := w.nestedExecution(parent, name, input)
	out, err := w.execute(e, input)
	if errors.Is(err, errSuspended) {
		return zero, err
	}
	if err != nil {
		e.journal.revertAll()
		state := TransactionStateReverted
		if e.journal.hasFailed {
			state = TransactionStateFailed
		}
		e.journal.setState(state)
		parent.journal.fail(rec, err)
		parent.mu.Lock()
		parent.errors = append(parent.errors, err)
		parent.mu.Unlock()
		// the failed step is compensated too: the nested transaction
		// already reverted itself
		parent.pushCompensation(compensation{name: name, run: func() error {
			parent.journal.compensateBegin(rec)
			parent.journal.compensated(rec, nil, false)
			return nil
		}})
		return zero, &stepFailed{err: err}
	}
	e.journal.skipUnreached()
	e.journal.setState(TransactionStateDone)
	var compInput any = parent.transactionID
	if !async {
		result := map[string]any{"errors": []any{}, "transaction": e.nestedTransaction(w.ID, input), "thrownError": nil}
		if !isNilOutput(any(out)) {
			result["result"] = out
		}
		compInput = result
	}
	// an undefined result leaves the step output undefined
	parent.journal.succeed(rec, out, !isNilOutput(any(out)), compInput, true)
	nested := e.compensations
	parent.pushCompensation(compensation{name: name, run: func() error {
		parent.journal.compensateBegin(rec)
		e.compensations = nested
		e.compensate()
		parent.journal.compensated(rec, nil, false)
		return nil
	}})
	return out, nil
}

// nestedExecution starts the execution of a workflow run as a step.
func (w *Workflow[I, O]) nestedExecution(parent *Execution, name string, input I) *Execution {
	j := newJournal(input)
	j.nested = true
	j.runID = parent.journal.runID
	j.nestedTxID = name + "-" + parent.transactionID
	j.parentStepKey = parent.workflowID + ":" + parent.transactionID + ":" + name + ":invoke"
	j.state = TransactionStateInvoking
	j.define(w.ID)
	return &Execution{ctx: parent.ctx, container: parent.container, workflowID: w.ID, transactionID: parent.transactionID, eventGroupID: parent.eventGroupID, hooks: w.hooks,
		journal: j, detacher: parent.detacher, rootWorkflowID: parent.rootWorkflowID, rootOptions: parent.rootOptions, subWorkflow: true,
		parentExec: parent}
}

// nestedTransaction mirrors the serialized DistributedTransaction of a
// workflow run as a step.
func (e *Execution) nestedTransaction(workflowID string, input any) map[string]any {
	flow := e.journal.flow(workflowID, e.transactionID, e.eventGroupID, WorkflowOptions{})
	ctx := e.journal.context()
	return map[string]any{
		"_events": map[string]any{}, "_eventsCount": 0, "flow": flow, "payload": input, "errors": ctx["errors"], "context": ctx["data"],
		"transactionId": e.journal.nestedTxID, "modelId": workflowID, "runId": e.journal.runID,
	}
}

// replayAsStep handles the "-as-step" step of a resumed run: a finished one
// returns its stored result and the nested run is rebuilt from the stored
// transaction (its steps restored, so it can be compensated).
func (w *Workflow[I, O]) replayAsStep(parent *Execution, name string, async bool, input I) (O, bool, error) {
	var zero O
	rp := parent.replay
	st := rp.take(name)
	if st == nil {
		return zero, false, nil
	}
	switch {
	case st.invokeState == stepStateDone:
		rec := parent.journal.restore(name, StepOptions{Async: async}, false, st)
		out, err := decodeStored[O](st.output)
		if err != nil {
			parent.journal.fail(rec, err)
			return zero, true, &stepFailed{err: err}
		}
		e := w.nestedExecution(parent, name, input)
		if tx := asRec(recGet(asRec(st.compInput), "transaction")); tx != nil {
			if flow := asRec(recGet(tx, "flow")); flow != nil {
				rs := parseCheckpoint(&Checkpoint{Flow: flow, Context: asRec(recGet(tx, "context"))})
				e.replay = newReplayState(rs, "", nil)
				if _, rerr := w.execute(e, input); rerr != nil {
					slog.Error(fmt.Sprintf("%s: cannot restore the nested transaction: %v", name, rerr))
				}
			}
		}
		nested := e.compensations
		parent.pushCompensation(compensation{name: name, run: func() error {
			parent.journal.compensateBegin(rec)
			e.compensations = nested
			e.compensate()
			parent.journal.compensated(rec, nil, false)
			return nil
		}})
		return out, true, nil
	case st.invokeState == stepStateNotStarted || st.invokeState == "":
		return zero, false, nil
	default:
		rp.stop()
		return zero, true, errSuspended
	}
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
		// an undefined result is not serialized
		if err == nil && !isNilOutput(out) {
			ev.Result, ev.HasResult = out, true
		}
		ev.Errors, ev.HasErrors = append([]any{}, e.journal.errors...), true
	})
}
