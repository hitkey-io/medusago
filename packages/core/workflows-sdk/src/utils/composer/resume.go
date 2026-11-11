package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// Durable runs (the Redis workflow engine).
//
// Upstream, a transaction waiting for setStepSuccess / setStepFailure lives
// only in its checkpoint (dtrx:<workflowId>:<transactionId> in Redis and
// the workflow_execution row): any instance loads it and continues the
// composed step tree. A Go workflow body is plain code, so a durable engine
// resumes it by deterministic replay: the body runs again from the start
// with the stored input; every step the checkpoint holds as done returns
// its stored output (decoded from JSON, like the parsed checkpoint
// upstream) without running and registers its compensation with the stored
// compensation input; the waiting step receives the delivered response;
// the steps after it run normally. A step the checkpoint holds as running,
// or waiting for another response, stops the replay. Steps are matched by
// action (step names are unique within a workflow), so a checkpoint stored
// by Node.js resumes the same way.
//
// With a durable engine a waiting step does not block its run: the
// checkpoint is persisted and the run stops; its step timeout is a job of
// the engine's queue.

// Checkpoint is a stored transaction: the TransactionFlow, the
// TransactionContext ({ invoke, payload, compensate }) and the errors.
type Checkpoint struct {
	Flow    *dal.Record
	Context *dal.Record
	Errors  []any
}

// StepTimeoutJob identifies the step timeout job of a waiting step
// (scheduleStepTimeout / clearStepTimeout).
type StepTimeoutJob struct {
	WorkflowID    string
	TransactionID string
	StepID        string
	// Metadata is the flow metadata (transactionMetadata of the job).
	Metadata *dal.Record
	// Seconds is the step timeout.
	Seconds int
}

// DurableEngine is implemented by a workflow engine module whose waiting
// steps outlive the process that ran them.
type DurableEngine interface {
	// LoadCheckpoint mirrors storage.get(key): nil when the transaction is
	// unknown or finished.
	LoadCheckpoint(ctx context.Context, workflowID, transactionID string) (*Checkpoint, error)
	// LockTransaction serializes the resumptions of a transaction.
	LockTransaction(ctx context.Context, workflowID, transactionID string) (unlock func(), err error)
	ScheduleStepTimeout(ctx context.Context, job StepTimeoutJob) error
	ClearStepTimeout(ctx context.Context, job StepTimeoutJob) error
}

func durableEngine(c Container) DurableEngine {
	if c == nil {
		return nil
	}
	d, _ := c.Resolve(utilsmodulessdk.WORKFLOW_ENGINE).(DurableEngine)
	return d
}

// errSuspended stops a workflow body at a waiting step of a durable run (or
// at a step a replay cannot pass). It is not a failure: nothing is
// compensated.
var errSuspended = errors.New("workflow execution suspended")

// storedStep is a step of a checkpoint.
type storedStep struct {
	id           string
	action       string
	uuid         string
	depth        float64
	invokeState  string
	invokeStatus string
	attempts     int
	failures     int
	startedAt    int64
	lastAttempt  int64
	timeout      float64
	output       any
	hasOutput    bool
	compInput    any
	hasComp      bool
	used         bool
}

func (st *storedStep) waiting() bool {
	return st.invokeState == stepStateInvoking && st.invokeStatus == stepStatusWaiting
}

// resumeState is a parsed checkpoint.
type resumeState struct {
	workflowID    string
	transactionID string
	runID         string
	eventGroupID  string
	startedAt     int64
	version       int
	hasAsync      bool
	payload       any
	invoke        map[string]any
	errors        []any
	options       *dal.Record
	steps         []*storedStep
}

func recGet(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func asRec(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

func asStr(v any) string {
	s, _ := v.(string)
	return s
}

func asNum(v any) float64 {
	f, _ := toFloat(v)
	return f
}

// parseCheckpoint reads a checkpoint stored by Go or by Node.js.
func parseCheckpoint(cp *Checkpoint) *resumeState {
	flow := cp.Flow
	data := asRec(recGet(cp.Context, "data"))
	if data == nil {
		data = cp.Context
	}
	rs := &resumeState{
		workflowID:    asStr(recGet(flow, "modelId")),
		transactionID: asStr(recGet(flow, "transactionId")),
		runID:         asStr(recGet(flow, "runId")),
		eventGroupID:  asStr(recGet(asRec(recGet(flow, "metadata")), "eventGroupId")),
		startedAt:     int64(asNum(recGet(flow, "startedAt"))),
		version:       int(asNum(recGet(flow, "_v"))),
		hasAsync:      recGet(flow, "hasAsyncSteps") == true,
		payload:       recGet(data, "payload"),
		invoke:        map[string]any{},
		errors:        append([]any{}, cp.Errors...),
		options:       asRec(recGet(flow, "options")),
	}
	if invoke := asRec(recGet(data, "invoke")); invoke != nil {
		for _, k := range invoke.Keys() {
			v, _ := invoke.Get(k)
			rs.invoke[k] = v
		}
	}
	if steps := asRec(recGet(flow, "steps")); steps != nil {
		for _, id := range steps.Keys() {
			if id == rootStepID {
				continue
			}
			s := asRec(recGet(steps, id))
			def := asRec(recGet(s, "definition"))
			inv := asRec(recGet(s, "invoke"))
			st := &storedStep{
				id: id, action: asStr(recGet(def, "action")), uuid: asStr(recGet(s, "uuid")), depth: asNum(recGet(s, "depth")),
				invokeState: asStr(recGet(inv, "state")), invokeStatus: asStr(recGet(inv, "status")),
				attempts: int(asNum(recGet(s, "attempts"))), failures: int(asNum(recGet(s, "failures"))),
				startedAt: int64(asNum(recGet(s, "startedAt"))), lastAttempt: int64(asNum(recGet(s, "lastAttempt"))),
				timeout: asNum(recGet(def, "timeout")),
			}
			if st.action == "" {
				continue
			}
			// context.invoke[action] = { __type, output: { __type, output,
			// compensateInput } }
			// (the response of setStepSuccess for an async step is stored
			// as is)
			if raw, ok := rs.invoke[st.action]; ok {
				env := asRec(raw)
				if env != nil && recGet(env, "__type") == symbolWorkflowData {
					resp := asRec(recGet(env, "output"))
					if resp != nil && recGet(resp, "__type") == symbolStepResponse {
						st.output, st.hasOutput = resp.Get("output")
						st.compInput, st.hasComp = resp.Get("compensateInput")
					} else {
						st.output, st.hasOutput = env.Get("output")
					}
				} else {
					st.output, st.hasOutput = raw, true
				}
			}
			rs.steps = append(rs.steps, st)
		}
	}
	sort.SliceStable(rs.steps, func(a, b int) bool {
		if rs.steps[a].depth != rs.steps[b].depth {
			return rs.steps[a].depth < rs.steps[b].depth
		}
		return rs.steps[a].startedAt < rs.steps[b].startedAt
	})
	return rs
}

func (rs *resumeState) stepByAction(action string) *storedStep {
	for _, st := range rs.steps {
		if st.action == action {
			return st
		}
	}
	return nil
}

// restoredJournal rebuilds the journal of a resumed transaction; its steps
// are appended as the replay passes them.
func restoredJournal(rs *resumeState) *journal {
	j := newJournal(rs.payload)
	if rs.runID != "" {
		j.runID = rs.runID
	}
	if rs.startedAt != 0 {
		j.startedAt = rs.startedAt
	}
	for k, v := range rs.invoke {
		j.invoke[k] = v
	}
	j.errors = rs.errors
	j.hasAsync = rs.hasAsync
	j.state = TransactionStateInvoking
	j.version = rs.version
	return j
}

// replayState drives the replay of a resumed run.
type replayState struct {
	mu        sync.Mutex
	steps     []*storedStep
	target    string
	response  *StepResult
	delivered bool
	stopped   bool
	signal    chan struct{}
	once      sync.Once
}

func newReplayState(rs *resumeState, target string, response *StepResult) *replayState {
	return &replayState{steps: rs.steps, target: target, response: response, signal: make(chan struct{})}
}

func (rp *replayState) take(action string) *storedStep {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	for _, st := range rp.steps {
		if !st.used && st.action == action {
			st.used = true
			return st
		}
	}
	return nil
}

func (rp *replayState) stop() {
	rp.mu.Lock()
	rp.stopped = true
	rp.mu.Unlock()
}

func (rp *replayState) isStopped() bool {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	return rp.stopped
}

// deliver hands out the response once.
func (rp *replayState) deliver(action string) *StepResult {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.delivered || rp.response == nil || action != rp.target {
		return nil
	}
	rp.delivered = true
	r := *rp.response
	return &r
}

// applied signals the caller of the resumption that the response was
// applied (or that the run went as far as it could).
func (rp *replayState) applied() { rp.once.Do(func() { close(rp.signal) }) }

// afterDelivered persists the step that received the delivered response
// (setStepSuccess / setStepFailure save the checkpoint) and signals it.
func (e *Execution) afterDelivered(r *stepRecord) {
	if e.replay == nil || r == nil || !r.delivered {
		return
	}
	r.delivered = false
	e.persist(TransactionStateInvoking)
	e.replay.applied()
}

// stepTimeoutJob describes the step timeout job of a waiting step.
func (e *Execution) stepTimeoutJob(r *stepRecord) StepTimeoutJob {
	md := dal.NewRecord()
	md.Set("eventGroupId", e.eventGroupID)
	md.Set("preventReleaseEvents", false)
	return StepTimeoutJob{WorkflowID: e.rootWorkflowID, TransactionID: e.transactionID, StepID: r.id, Metadata: md, Seconds: r.timeout}
}

// decodeStored converts a value of a checkpoint (JSON) into the step output
// type.
func decodeStored[O any](v any) (O, error) {
	var out O
	if v == nil {
		return out, nil
	}
	if o, ok := v.(O); ok {
		return o, nil
	}
	if err := DecodeInput(v, &out); err == nil {
		return out, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return out, err
	}
	var alt O
	if jerr := json.Unmarshal(b, &alt); jerr != nil {
		return alt, fmt.Errorf("cannot restore the stored value: %w", jerr)
	}
	return alt, nil
}

func sameJSON(a, b any) bool {
	ja, err := json.Marshal(a)
	if err != nil {
		return false
	}
	jb, err := json.Marshal(b)
	return err == nil && string(ja) == string(jb)
}

// replayStep handles a step of a resumed run: handled is false when the
// step runs normally.
func (s *Step[I, O]) replayStep(e *Execution, action string, noComp bool) (O, bool, error) {
	var zero O
	rp := e.replay
	st := rp.take(action)
	if st == nil {
		return zero, false, nil
	}
	switch {
	case st.invokeState == stepStateDone:
		r := e.journal.restore(action, s.Options, noComp, st)
		var out O
		if st.hasOutput {
			var err error
			out, err = decodeStored[O](st.output)
			if err != nil {
				out, ferr := s.settle(e, r, StepResponse[O]{}, err)
				return out, true, ferr
			}
		}
		// the compensation input defaults to the output: then it gets the
		// output restored with its type
		var compInput any = out
		if st.hasComp && !sameJSON(st.compInput, st.output) {
			compInput = st.compInput
		}
		s.pushStepCompensation(e, r, compInput)
		return out, true, nil
	case st.invokeState == stepStateSkipped:
		e.journal.restore(action, s.Options, noComp, st)
		if !e.subWorkflow {
			e.journal.mu.Lock()
			e.journal.hasSkipped = true
			e.journal.mu.Unlock()
		}
		return zero, true, nil
	case st.waiting() && action == rp.target:
		resp := rp.deliver(action)
		if resp == nil {
			rp.stop()
			return zero, true, errSuspended
		}
		r := e.journal.restore(action, s.Options, noComp, st)
		r.delivered = true
		if s.Options.Timeout > 0 {
			if de := durableEngine(e.container); de != nil {
				if err := de.ClearStepTimeout(context.WithoutCancel(e.ctx), e.stepTimeoutJob(r)); err != nil {
					slog.Error(fmt.Sprintf("failed to clear the timeout of step %s: %v", r.id, err))
				}
			}
		}
		res, err := applyStepResult[O](*resp)
		if resp.Err == nil {
			r.raw, r.hasRaw = resp.Output, true
		}
		out, ferr := s.settle(e, r, res, err)
		return out, true, ferr
	case (st.invokeState == stepStateFailed || st.invokeState == stepStateTimeout) && s.Options.ContinueOnPermanentFailure:
		e.journal.restore(action, s.Options, noComp, st)
		return zero, true, nil
	case st.invokeState == stepStateNotStarted || st.invokeState == "":
		// not run yet (a Node.js flow lists every step)
		return zero, false, nil
	default:
		// running or waiting elsewhere: the replay cannot pass it
		rp.stop()
		return zero, true, errSuspended
	}
}

// resumer resumes a registered workflow from a parsed checkpoint.
type resumer func(ctx context.Context, c Container, rs *resumeState, rp *replayState) (*resumedRun, error)

var resumers = map[string]resumer{}

// resumedRun is a resumption in progress.
type resumedRun struct {
	detached <-chan struct{}
	finished chan struct{}
}

// wait returns once the response was applied, or the run detached
// (background step, next waiting step) or finished.
func (r *resumedRun) wait(rp *replayState) {
	select {
	case <-rp.signal:
	case <-r.detached:
	case <-r.finished:
	}
}

func registerResumer[I, O any](w *Workflow[I, O]) {
	resumers[w.ID] = func(ctx context.Context, c Container, rs *resumeState, rp *replayState) (*resumedRun, error) {
		return w.resume(ctx, c, rs, rp)
	}
}

// resume runs the body of a stored transaction again (see the notes on
// durable runs above).
func (w *Workflow[I, O]) resume(ctx context.Context, c Container, rs *resumeState, rp *replayState) (*resumedRun, error) {
	in, err := decodeStored[I](rs.payload)
	if err != nil {
		return nil, err
	}
	e := &Execution{ctx: context.WithoutCancel(ctx), container: c, workflowID: w.ID, transactionID: rs.transactionID, eventGroupID: rs.eventGroupID,
		hooks: w.hooks, journal: restoredJournal(rs), detacher: newDetacher(), rootWorkflowID: w.ID, rootOptions: w.Options, replay: rp}
	e.persistedRev = e.journal.rev
	e.notifyEvents()
	e.journal.emitRun("onResume", nil)
	run := &resumedRun{detached: e.detacher.ch, finished: make(chan struct{})}
	go func() {
		defer close(run.finished)
		out, err := w.execute(e, in)
		if errors.Is(err, errSuspended) {
			if !rp.isStopped() {
				e.persistIfChanged(TransactionStateInvoking)
			}
			return
		}
		w.finish(e, err, true)
		e.emitFinish(out, err)
	}()
	return run, nil
}

// deliverDurable mirrors registerStepSuccess / registerStepFailure on a
// durable engine: the checkpoint is loaded and the transaction resumed in
// this instance with the response.
func deliverDurable(ctx context.Context, c Container, de DurableEngine, key IdempotencyKey, r StepResult, failure bool) error {
	if !IsRegisteredWorkflow(key.WorkflowID) {
		return fmt.Errorf("Workflow with id \"%s\" not found.", key.WorkflowID)
	}
	unlock, err := de.LockTransaction(ctx, key.WorkflowID, key.TransactionID)
	if err != nil {
		return err
	}
	defer unlock()
	cp, err := de.LoadCheckpoint(ctx, key.WorkflowID, key.TransactionID)
	if err != nil {
		return err
	}
	if cp == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Transaction %s could not be found.", key.TransactionID))
	}
	rs := parseCheckpoint(cp)
	st := rs.stepByAction(key.StepID)
	if st == nil {
		return errors.New("Action not found.")
	}
	if key.Action != "" && key.Action != "invoke" {
		return errors.New("Incorrect action type.")
	}
	if !st.waiting() {
		verb := "success"
		if failure {
			verb = "failure"
		}
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Cannot set step %s when status is %s", verb, st.invokeStatus))
	}
	return startResume(ctx, c, rs, newReplayState(rs, st.action, &r))
}

func startResume(ctx context.Context, c Container, rs *resumeState, rp *replayState) error {
	registryMu.RLock()
	resume := resumers[rs.workflowID]
	registryMu.RUnlock()
	if resume == nil {
		return notFoundWorkflow(rs.workflowID)
	}
	run, err := resume(ctx, c, rs, rp)
	if err != nil {
		return err
	}
	run.wait(rp)
	return nil
}

// ResumeTransaction mirrors the workflow engine worker running a job of
// its queue (retry, step_timeout, transaction_timeout): the stored
// transaction continues in this instance. A waiting step whose timeout
// expired fails with a TransactionStepTimeoutError, one of a transaction
// whose timeout expired with a TransactionTimeoutError; a transaction
// without a waiting step continues after its finished steps. A finished or
// unknown transaction, or one still waiting for a response, is skipped.
func ResumeTransaction(ctx context.Context, c Container, workflowID, transactionID string) error {
	if !IsRegisteredWorkflow(workflowID) {
		return notFoundWorkflow(workflowID)
	}
	de := durableEngine(c)
	if de == nil {
		return nil
	}
	unlock, err := de.LockTransaction(ctx, workflowID, transactionID)
	if err != nil {
		return err
	}
	defer unlock()
	cp, err := de.LoadCheckpoint(ctx, workflowID, transactionID)
	if err != nil || cp == nil {
		return err
	}
	rs := parseCheckpoint(cp)
	now := time.Now().UnixMilli()
	var target string
	var response *StepResult
	waiting := false
	transactionTimeout := asNum(recGet(rs.options, "timeout"))
	for _, st := range rs.steps {
		if !st.waiting() {
			continue
		}
		waiting = true
		if st.timeout > 0 && st.startedAt+int64(st.timeout*1000) <= now {
			target, response = st.action, &StepResult{Err: &TransactionStepTimeoutError{}}
			break
		}
		if transactionTimeout > 0 && rs.startedAt+int64(transactionTimeout*1000) <= now {
			target, response = st.action, &StepResult{Err: &TransactionTimeoutError{}}
			break
		}
	}
	if waiting && response == nil {
		return nil
	}
	return startResume(ctx, c, rs, newReplayState(rs, target, response))
}

// TransactionTimeoutError mirrors TransactionTimeoutError.
type TransactionTimeoutError struct{}

func (*TransactionTimeoutError) Error() string { return "Timeout error" }
