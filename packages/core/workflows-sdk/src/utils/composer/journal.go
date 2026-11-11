package composer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// TransactionState mirrors TransactionState.
const (
	TransactionStateNotStarted          = "not_started"
	TransactionStateInvoking            = "invoking"
	TransactionStateWaitingToCompensate = "waiting_to_compensate"
	TransactionStateCompensating        = "compensating"
	TransactionStateDone                = "done"
	TransactionStateReverted            = "reverted"
	TransactionStateFailed              = "failed"
)

// TransactionStepState / TransactionStepStatus mirror the step states.
const (
	stepStateInvoking     = "invoking"
	stepStateDone         = "done"
	stepStateFailed       = "failed"
	stepStateDormant      = "dormant"
	stepStateReverted     = "reverted"
	stepStateSkipped      = "skipped"
	stepStateTimeout      = "timeout"
	stepStateNotStarted   = "not_started"
	stepStatusIdle        = "idle"
	stepStatusOK          = "ok"
	stepStatusWaiting     = "waiting_response"
	stepStatusPermanent   = "permanent_failure"
	symbolWorkflowData    = "Symbol(WorkflowWorkflowData)"
	symbolStepResponse    = "Symbol(WorkflowStepResponse)"
	rootStepID            = "_root"
	stepIDSeparator       = "."
	maxStepIDDisambiguate = 1000
)

// stepRecord is one entry of TransactionFlow.steps.
type stepRecord struct {
	id              string
	action          string
	uuid            string
	depth           int
	next            []string
	invokeState     string
	invokeStatus    string
	compState       string
	compStatus      string
	attempts        int
	failures        int
	startedAt       int64
	lastAttempt     int64
	async           bool
	compensateAsync bool
	store           bool
	noCompensation  bool
	stepFailed      bool
	// timeout is the step timeout in seconds (definition.timeout)
	timeout int
	// delivered: the step receives the response of a resumption
	delivered bool
}

// journal records a run the way the TransactionOrchestrator checkpoints it:
// the executed steps as a TransactionFlow (ids "_root.<a>.<b>", depth,
// invoke/compensate states) and the TransactionContext (step outputs keyed
// by action, payload, errors). Upstream builds the flow from the composed
// step tree; here it is built from the steps as they run, so branches that
// did not run (when()) are absent — exactly like upstream, which omits the
// steps of conditions that were not met — while parallel branches are
// chained in their execution order.
type journal struct {
	mu        sync.Mutex
	runID     string
	startedAt int64
	steps     []*stepRecord
	byID      map[string]*stepRecord
	root      *stepRecord
	tip       *stepRecord
	invoke    map[string]any
	payload   any
	errors    []any
	hasAsync  bool
	hasFailed bool
	// hasSkipped: a step returned StepResponse.skip()
	hasSkipped bool
	waiting    bool
	state      string
	version    int
	// rev counts the changes (persistIfChanged)
	rev int
	// events receives the run's notifications (nil: not notified).
	events func(WorkflowEvent)
}

func nowMillis() int64 { return time.Now().UnixMilli() }

func newJournal(payload any) *journal {
	root := &stepRecord{id: rootStepID, next: []string{}}
	return &journal{
		runID:     ulid.Make().String(),
		startedAt: nowMillis(),
		byID:      map[string]*stepRecord{rootStepID: root},
		root:      root,
		tip:       root,
		invoke:    map[string]any{},
		payload:   payload,
		errors:    []any{},
		state:     TransactionStateNotStarted,
	}
}

// nextID derives the id of a step appended after the tip
// ("<parent>.<action>", disambiguated when the action repeats).
func (j *journal) nextID(parent *stepRecord, action string) string {
	id := parent.id + stepIDSeparator + action
	if _, exists := j.byID[id]; exists {
		for n := 1; n < maxStepIDDisambiguate; n++ {
			candidate := fmt.Sprintf("%s-%d", id, n)
			if _, taken := j.byID[candidate]; !taken {
				id = candidate
				break
			}
		}
	}
	return id
}

func (j *journal) push(parent, r *stepRecord) {
	j.rev++
	if r.async {
		j.hasAsync = true
	}
	parent.next = append(parent.next, r.id)
	j.steps = append(j.steps, r)
	j.byID[r.id] = r
	j.tip = r
}

// begin appends a step after the current tip and makes it the new tip.
func (j *journal) begin(action string, opts StepOptions, noCompensation bool) *stepRecord {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.state == TransactionStateNotStarted {
		j.state = TransactionStateInvoking
	}
	parent := j.tip
	now := nowMillis()
	r := &stepRecord{
		id: j.nextID(parent, action), action: action, uuid: ulid.Make().String(), depth: parent.depth + 1, next: []string{},
		invokeState: stepStateInvoking, invokeStatus: stepStatusIdle, compState: stepStateDormant, compStatus: stepStatusIdle,
		attempts: 1, startedAt: now, lastAttempt: now, async: opts.Async, store: opts.Store, noCompensation: noCompensation,
		timeout: opts.Timeout,
	}
	j.push(parent, r)
	j.emit("onStepBegin", r, nil)
	return r
}

// restore appends a step of a resumed transaction with the state stored in
// its checkpoint (no notification: the step ran before the resumption).
func (j *journal) restore(action string, opts StepOptions, noCompensation bool, st *storedStep) *stepRecord {
	j.mu.Lock()
	defer j.mu.Unlock()
	parent := j.tip
	uuid := st.uuid
	if uuid == "" {
		uuid = ulid.Make().String()
	}
	r := &stepRecord{
		id: j.nextID(parent, action), action: action, uuid: uuid, depth: parent.depth + 1, next: []string{},
		invokeState: st.invokeState, invokeStatus: st.invokeStatus, compState: stepStateDormant, compStatus: stepStatusIdle,
		attempts: st.attempts, failures: st.failures, startedAt: st.startedAt, lastAttempt: st.lastAttempt,
		async: opts.Async, store: opts.Store, noCompensation: noCompensation, timeout: opts.Timeout,
		stepFailed: st.invokeState == stepStateFailed || st.invokeState == stepStateTimeout,
	}
	j.push(parent, r)
	return r
}

func stepResponseEnvelope(output any, hasOutput bool, compInput any, hasComp bool) map[string]any {
	resp := map[string]any{"__type": symbolStepResponse}
	if hasOutput {
		resp["output"] = output
	}
	if hasComp {
		resp["compensateInput"] = compInput
	}
	return map[string]any{"__type": symbolWorkflowData, "output": resp}
}

func (j *journal) succeed(r *stepRecord, output any, hasOutput bool, compInput any, hasComp bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	r.invokeState, r.invokeStatus = stepStateDone, stepStatusOK
	r.lastAttempt = nowMillis()
	j.invoke[r.action] = stepResponseEnvelope(output, hasOutput, compInput, hasComp)
	j.emit("onStepSuccess", r, func(ev *WorkflowEvent) { ev.Response, ev.HasResponse = output, hasOutput })
}

// skip mirrors skipStep: the step is SKIPPED with status OK. flagFlow sets
// hasSkippedSteps (steps of workflows run as a step belong to their own
// flow upstream, they do not flag the root one).
func (j *journal) skip(r *stepRecord, flagFlow bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	r.invokeState, r.invokeStatus = stepStateSkipped, stepStatusOK
	if flagFlow {
		j.hasSkipped = true
	}
	r.lastAttempt = nowMillis()
}

func (j *journal) wait(r *stepRecord) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	r.invokeState, r.invokeStatus = stepStateInvoking, stepStatusWaiting
	j.waiting = true
	j.emit("onStepAwaiting", r, nil)
}

func (j *journal) attempt(r *stepRecord, n int) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	r.attempts = n
	r.lastAttempt = nowMillis()
}

// serializeError mirrors the errors stored in the checkpoint: a
// MedusaError keeps its type, date and __isMedusaError flag.
func serializeError(err error) map[string]any {
	out := map[string]any{"message": err.Error(), "name": "Error", "stack": "Error: " + err.Error()}
	var me *common.MedusaError
	if errors.As(err, &me) {
		out["type"] = me.Type
		out["date"] = me.Date.UTC().Format("2006-01-02T15:04:05.000Z")
		out["__isMedusaError"] = true
		if me.Code != "" {
			out["code"] = me.Code
		}
	}
	return out
}

func (j *journal) fail(r *stepRecord, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	r.invokeState, r.invokeStatus = stepStateFailed, stepStatusPermanent
	var timeout *TransactionStepTimeoutError
	if errors.As(err, &timeout) {
		// setStepTimeout: the step state is TIMEOUT
		r.invokeState = stepStateTimeout
	}
	r.stepFailed = true
	r.lastAttempt = nowMillis()
	j.errors = append(j.errors, map[string]any{"action": r.action, "handlerType": "invoke", "error": serializeError(err)})
	errs := j.errorsFor(r.action, "invoke")
	j.emit("onStepFailure", r, func(ev *WorkflowEvent) { ev.Errors, ev.HasErrors = errs, true })
}

// revertAll mirrors the flow after compensation: every step that ran is
// reverted (status ok when its compensation ran, idle otherwise) and
// flagged stepFailed.
func (j *journal) revertAll() {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, r := range j.steps {
		if r.invokeState != stepStateDone && r.invokeState != stepStateFailed && r.invokeState != stepStateTimeout {
			continue
		}
		if r.compState == stepStateDormant {
			r.compState, r.compStatus = stepStateReverted, stepStatusIdle
		}
		r.stepFailed = true
	}
}

func (j *journal) compensated(r *stepRecord, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	if err != nil {
		r.compState, r.compStatus = stepStateFailed, stepStatusPermanent
		j.hasFailed = true
		j.errors = append(j.errors, map[string]any{"action": r.action, "handlerType": "compensate", "error": serializeError(err)})
		errs := j.errorsFor(r.action, "compensate")
		j.emit("onCompensateStepFailure", r, func(ev *WorkflowEvent) { ev.Errors, ev.HasErrors = errs, true })
		return
	}
	r.compState, r.compStatus = stepStateReverted, stepStatusOK
	j.emit("onCompensateStepSuccess", r, nil)
}

// emitRun sends a run-level event (onBegin, onCompensateBegin, onFinish).
func (j *journal) emitRun(eventType string, fill func(*WorkflowEvent)) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.emit(eventType, nil, fill)
}

func (j *journal) setState(state string) {
	j.mu.Lock()
	j.state = state
	j.mu.Unlock()
}

func (j *journal) currentState() string {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.state
}

// definitionTree mirrors flow.definition: the step definitions nested by
// "next" (a list for parallel branches).
func (j *journal) definitionTree(r *stepRecord) any {
	var children []any
	for _, id := range r.next {
		c := j.byID[id]
		d := map[string]any{"uuid": c.uuid, "action": c.action, "noCompensation": c.noCompensation}
		if c.async {
			d["async"] = true
		}
		if c.timeout > 0 {
			d["timeout"] = c.timeout
		}
		if sub := j.definitionTree(c); sub != nil {
			d["next"] = sub
		}
		children = append(children, d)
	}
	switch len(children) {
	case 0:
		return nil
	case 1:
		return children[0]
	default:
		return children
	}
}

// flow mirrors TransactionFlow as stored in workflow_execution.execution.
func (j *journal) flow(workflowID, transactionID, eventGroupID string, opts WorkflowOptions) map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	steps := map[string]any{rootStepID: map[string]any{"id": rootStepID, "next": append([]string{}, j.root.next...)}}
	hasReverted := false
	for _, r := range j.steps {
		def := map[string]any{"uuid": r.uuid, "store": r.store, "action": r.action, "noCompensation": r.noCompensation}
		if r.async {
			def["async"] = true
			def["compensateAsync"] = r.compensateAsync
		}
		if r.timeout > 0 {
			def["timeout"] = r.timeout
		}
		if r.compState == stepStateReverted {
			hasReverted = true
		}
		steps[r.id] = map[string]any{
			"_v": j.version, "id": r.id, "next": append([]string{}, r.next...), "uuid": r.uuid, "depth": r.depth,
			"invoke":     map[string]any{"state": r.invokeState, "status": r.invokeStatus},
			"compensate": map[string]any{"state": r.compState, "status": r.compStatus},
			"attempts":   r.attempts, "failures": r.failures, "startedAt": r.startedAt, "lastAttempt": r.lastAttempt,
			"definition": def, "stepFailed": r.stepFailed, "saveResponse": true,
		}
	}
	options := map[string]any{"name": workflowID, "store": opts.Store, "idempotent": opts.Idempotent}
	if opts.RetentionTime > 0 {
		options["retentionTime"] = opts.RetentionTime
	}
	definition := j.definitionTree(j.root)
	if definition == nil {
		definition = map[string]any{}
	}
	return map[string]any{
		"_v": j.version, "runId": j.runID, "state": j.state, "steps": steps, "modelId": workflowID, "options": options,
		"metadata":  map[string]any{"eventGroupId": eventGroupID, "preventReleaseEvents": false},
		"startedAt": j.startedAt, "definition": definition, "timedOutAt": nil, "hasAsyncSteps": j.hasAsync,
		"transactionId": transactionID, "hasFailedSteps": j.hasFailed, "hasSkippedSteps": j.hasSkipped, "hasWaitingSteps": j.waiting,
		"hasRevertedSteps": hasReverted, "hasSkippedOnFailureSteps": false,
	}
}

// context mirrors { data: TransactionContext, errors } as stored in
// workflow_execution.context.
func (j *journal) context() map[string]any {
	j.mu.Lock()
	defer j.mu.Unlock()
	invoke := make(map[string]any, len(j.invoke))
	for k, v := range j.invoke {
		invoke[k] = v
	}
	return map[string]any{
		"data":   map[string]any{"invoke": invoke, "payload": j.payload, "compensate": map[string]any{}},
		"errors": append([]any{}, j.errors...),
	}
}
