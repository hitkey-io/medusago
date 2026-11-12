package composer

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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
	stepStateCompensating = "compensating"
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
	// raw is the response registered by setStepSuccess for an async step
	// (stored as is in the context, without the step response envelope)
	raw    any
	hasRaw bool
	// predefined: the step comes from the graph upstream composes; fields
	// are its definition fields (action, noCompensation, async, ...)
	predefined bool
	fields     *dal.Record
	// retryReset: a failure reset retryRescheduledAt (the property exists,
	// null)
	retryReset bool
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
	mu         sync.Mutex
	runID      string
	startedAt  int64
	steps      []*stepRecord
	byID       map[string]*stepRecord
	root       *stepRecord
	tip        *stepRecord
	invoke     map[string]any
	compensate map[string]any
	payload    any
	errors     []any
	hasAsync   bool
	hasFailed  bool
	// hasSkipped: a step returned StepResponse.skip()
	hasSkipped bool
	waiting    bool
	state      string
	version    int
	// rev counts the changes (persistIfChanged)
	rev int
	// events receives the run's notifications (nil: not notified).
	events func(WorkflowEvent)
	// defined: the steps follow the graph upstream composes for the
	// workflow; pending lists the steps of an action that did not run yet
	defined bool
	pending map[string][]*stepRecord
	// nested: the journal of a workflow run as a step (its flow has empty
	// options and the metadata of the parent step)
	nested        bool
	parentStepKey string
	nestedTxID    string
	// flowOptions are the options the workflow is created with upstream
	// (nil when it has no upstream counterpart)
	flowOptions *dal.Record
	// engineRun: the run goes through the workflow engine (its metadata
	// carries preventReleaseEvents: false)
	engineRun bool
}

func nowMillis() int64 { return time.Now().UnixMilli() }

func newJournal(payload any) *journal {
	root := &stepRecord{id: rootStepID, next: []string{}}
	return &journal{
		runID:      ulid.Make().String(),
		startedAt:  nowMillis(),
		byID:       map[string]*stepRecord{rootStepID: root},
		root:       root,
		tip:        root,
		invoke:     map[string]any{},
		compensate: map[string]any{},
		payload:    payload,
		errors:     []any{},
		state:      TransactionStateNotStarted,
	}
}

// define lays out the steps of the graph upstream composes for the
// workflow: they are listed from the start (not started), each run step
// takes its place.
func (j *journal) define(workflowID string) {
	d := lookupFlowDef(workflowID)
	if d == nil || len(d.nodes) == 0 {
		return
	}
	nodes := d.nodes
	j.mu.Lock()
	defer j.mu.Unlock()
	j.defined = true
	j.flowOptions = d.options
	j.pending = map[string][]*stepRecord{}
	var add func(parent *stepRecord, ns []*flowNode)
	add = func(parent *stepRecord, ns []*flowNode) {
		for _, n := range ns {
			r := &stepRecord{
				id: parent.id + stepIDSeparator + n.action, action: n.action, uuid: n.uuid, depth: parent.depth + 1, next: []string{},
				invokeState: stepStateNotStarted, invokeStatus: stepStatusIdle, compState: stepStateDormant, compStatus: stepStatusIdle,
				async: fieldBool(n.fields, "async"), compensateAsync: fieldBool(n.fields, "compensateAsync"),
				noCompensation: fieldBool(n.fields, "noCompensation"), timeout: fieldInt(n.fields, "timeout"),
				predefined: true, fields: n.fields,
			}
			if r.async {
				j.hasAsync = true
			}
			parent.next = append(parent.next, r.id)
			j.steps = append(j.steps, r)
			j.byID[r.id] = r
			j.pending[n.action] = append(j.pending[n.action], r)
			add(r, n.next)
		}
	}
	add(j.root, nodes)
}

// takePending returns the step of the graph with the action that did not
// run yet (nil when the graph has none).
func (j *journal) takePending(action string) *stepRecord {
	if !j.defined {
		return nil
	}
	list := j.pending[action]
	if len(list) == 0 {
		return nil
	}
	j.pending[action] = list[1:]
	return list[0]
}

// skipUnreached mirrors the steps of a when() whose condition was false:
// a finished run marks the steps of its graph that did not run skipped.
func (j *journal) skipUnreached() {
	j.mu.Lock()
	defer j.mu.Unlock()
	now := nowMillis()
	for _, r := range j.steps {
		if !r.predefined || r.invokeState != stepStateNotStarted {
			continue
		}
		r.invokeState, r.invokeStatus = stepStateSkipped, stepStatusOK
		r.attempts, r.startedAt, r.lastAttempt = 1, now, now
		j.hasSkipped = true
		j.rev++
		// the orchestrator begins the step before its condition skips it
		j.emit("onStepBegin", r, nil)
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
	now := nowMillis()
	if r := j.takePending(action); r != nil {
		j.rev++
		// a running step waits for its handler's response
		r.invokeState, r.invokeStatus = stepStateInvoking, stepStatusWaiting
		r.attempts, r.startedAt, r.lastAttempt = 1, now, now
		r.store = opts.Store
		if opts.Async {
			r.async = true
			j.hasAsync = true
		}
		if opts.Timeout > 0 {
			r.timeout = opts.Timeout
		}
		j.tip = r
		j.emit("onStepBegin", r, nil)
		return r
	}
	parent := j.tip
	r := &stepRecord{
		id: j.nextID(parent, action), action: action, uuid: ulid.Make().String(), depth: parent.depth + 1, next: []string{},
		invokeState: stepStateInvoking, invokeStatus: stepStatusWaiting, compState: stepStateDormant, compStatus: stepStatusIdle,
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
	if r := j.takePending(action); r != nil {
		if st.uuid != "" {
			r.uuid = st.uuid
		}
		r.invokeState, r.invokeStatus = st.invokeState, st.invokeStatus
		r.attempts, r.failures, r.startedAt, r.lastAttempt = st.attempts, st.failures, st.startedAt, st.lastAttempt
		r.store = opts.Store
		if opts.Async {
			r.async = true
		}
		if opts.Timeout > 0 {
			r.timeout = opts.Timeout
		}
		r.stepFailed = st.invokeState == stepStateFailed || st.invokeState == stepStateTimeout
		j.tip = r
		return r
	}
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
	switch {
	case r.hasRaw:
		j.invoke[r.action] = r.raw
	default:
		if !hasComp && hasOutput {
			// new StepResponse(output): compensateInput defaults to the output
			compInput, hasComp = output, true
		}
		j.invoke[r.action] = stepResponseEnvelope(output, hasOutput, compInput, hasComp)
	}
	j.emit("onStepSuccess", r, func(ev *WorkflowEvent) { ev.Response, ev.HasResponse = output, hasOutput })
}

// succeedRaw records a step whose handler returned a value that is not a
// StepResponse: the context holds it as the output as is.
func (j *journal) succeedRaw(r *stepRecord, output any) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	r.invokeState, r.invokeStatus = stepStateDone, stepStatusOK
	r.lastAttempt = nowMillis()
	j.invoke[r.action] = map[string]any{"__type": symbolWorkflowData, "output": output}
	j.emit("onStepSuccess", r, func(ev *WorkflowEvent) { ev.Response, ev.HasResponse = output, true })
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
	if n > 1 {
		// the previous attempt failed (handleStepFailure)
		r.retryReset = true
	}
}

// serializeError mirrors the errors stored in the checkpoint: a
// MedusaError keeps its type, date and __isMedusaError flag (in the order
// of the serialized error: message, name, stack, then the own properties).
func serializeError(err error) *dal.Record {
	out := dal.NewRecord()
	out.Set("message", err.Error())
	out.Set("name", "Error")
	out.Set("stack", "Error: "+err.Error())
	var me *common.MedusaError
	if errors.As(err, &me) {
		out.Set("__isMedusaError", true)
		out.Set("type", me.Type)
		if me.Code != "" {
			out.Set("code", me.Code)
		}
		out.Set("date", me.Date.UTC().Format("2006-01-02T15:04:05.000Z"))
	}
	return out
}

// stepError mirrors a TransactionStepError: { action, handlerType, error }.
func stepError(action, handlerType string, err error) *dal.Record {
	out := dal.NewRecord()
	out.Set("action", action)
	out.Set("handlerType", handlerType)
	out.Set("error", serializeError(err))
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
	r.retryReset = true
	r.lastAttempt = nowMillis()
	j.errors = append(j.errors, stepError(r.action, "invoke", err))
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

// compensateBegin mirrors a step starting its compensation: COMPENSATING,
// waiting for the handler, onStepBegin.
func (j *journal) compensateBegin(r *stepRecord) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	r.compState, r.compStatus = stepStateCompensating, stepStatusWaiting
	j.emit("onStepBegin", r, nil)
}

func (j *journal) compensated(r *stepRecord, err error, response bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.rev++
	if err != nil {
		r.compState, r.compStatus = stepStateFailed, stepStatusPermanent
		j.hasFailed = true
		j.errors = append(j.errors, stepError(r.action, "compensate", err))
		errs := j.errorsFor(r.action, "compensate")
		j.emit("onCompensateStepFailure", r, func(ev *WorkflowEvent) { ev.Errors, ev.HasErrors = errs, true })
		return
	}
	r.compState, r.compStatus = stepStateReverted, stepStatusOK
	// the compensation handler's { output } (undefined unless it returned
	// a StepResponse)
	entry := map[string]any{}
	if response {
		entry["output"] = map[string]any{"__type": symbolStepResponse}
	}
	j.compensate[r.action] = entry
	j.emit("onCompensateStepSuccess", r, func(ev *WorkflowEvent) { ev.Response, ev.HasResponse = entry, true })
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
		var d map[string]any
		if c.fields != nil {
			d = recordMap(c.fields)
			d["uuid"] = c.uuid
		} else {
			d = map[string]any{"uuid": c.uuid, "action": c.action, "noCompensation": c.noCompensation}
			if c.async {
				d["async"] = true
			}
			if c.timeout > 0 {
				d["timeout"] = c.timeout
			}
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
		def := r.definition()
		if r.compState == stepStateReverted {
			hasReverted = true
		}
		step := map[string]any{
			"_v": j.version, "id": r.id, "next": append([]string{}, r.next...), "uuid": r.uuid, "depth": r.depth,
			"invoke":     map[string]any{"state": r.invokeState, "status": r.invokeStatus},
			"compensate": map[string]any{"state": r.compState, "status": r.compStatus},
			"attempts":   r.attempts, "failures": r.failures, "startedAt": r.startedAt, "lastAttempt": r.lastAttempt,
			"definition": def, "stepFailed": r.stepFailed, "saveResponse": true,
		}
		if r.retryReset {
			step["retryRescheduledAt"] = nil
		}
		if r.invokeState == stepStateNotStarted && r.attempts == 0 {
			// a step that never ran has no startedAt and a null lastAttempt
			delete(step, "startedAt")
			step["lastAttempt"] = nil
		}
		steps[r.id] = step
	}
	options := map[string]any{"name": workflowID, "store": opts.Store, "idempotent": opts.Idempotent}
	if opts.RetentionTime > 0 {
		options["retentionTime"] = opts.RetentionTime
	}
	if j.flowOptions != nil {
		options = j.parsedFlowOptions()
	}
	// the metadata of the run: preventReleaseEvents is set by the workflow
	// engine (false) and for a workflow run as a step (true)
	metadata := map[string]any{"eventGroupId": eventGroupID}
	if j.engineRun {
		metadata["preventReleaseEvents"] = false
	}
	if j.nested {
		metadata["parentStepIdempotencyKey"] = j.parentStepKey
		metadata["preventReleaseEvents"] = true
		transactionID = j.nestedTxID
	}
	definition := j.definitionTree(j.root)
	if definition == nil {
		definition = map[string]any{}
	}
	return map[string]any{
		"_v": j.version, "runId": j.runID, "state": j.state, "steps": steps, "modelId": workflowID, "options": options,
		"metadata":  metadata,
		"startedAt": j.startedAt, "definition": definition, "timedOutAt": nil, "hasAsyncSteps": j.hasAsync,
		"transactionId": transactionID, "hasFailedSteps": j.hasFailed, "hasSkippedSteps": j.hasSkipped, "hasWaitingSteps": j.waiting,
		"hasRevertedSteps": hasReverted, "hasSkippedOnFailureSteps": false,
	}
}

// parsedFlowOptions mirrors parseFlowOptions: the options the workflow is
// created with, store forced when a step is async, has a timeout or a retry
// interval, or the workflow has a timeout, is idempotent or retained
// (caller holds j.mu).
func (j *journal) parsedFlowOptions() map[string]any {
	options := recordMap(j.flowOptions)
	force := j.hasAsync || fieldBool(j.flowOptions, "idempotent") || fieldInt(j.flowOptions, "retentionTime") > 0 || fieldInt(j.flowOptions, "timeout") > 0
	for _, r := range j.steps {
		if r.async || r.timeout > 0 || fieldInt(r.fields, "retryInterval") > 0 {
			force = true
		}
	}
	if force {
		options["store"] = true
	}
	return options
}

// definition mirrors step.definition: the definition fields of the step,
// its uuid and store (caller holds j.mu).
func (r *stepRecord) definition() map[string]any {
	if r.fields != nil {
		def := recordMap(r.fields)
		def["uuid"] = r.uuid
		// buildSteps forces store for async steps and steps with a timeout
		// or a retry interval
		def["store"] = fieldBool(r.fields, "store") || r.store || r.async || r.timeout > 0 || fieldInt(r.fields, "retryInterval") > 0
		if r.async {
			def["async"] = true
		}
		if r.timeout > 0 {
			def["timeout"] = r.timeout
		}
		return def
	}
	def := map[string]any{"uuid": r.uuid, "store": r.store, "action": r.action, "noCompensation": r.noCompensation}
	if r.async {
		def["async"] = true
		def["compensateAsync"] = r.compensateAsync
	}
	if r.timeout > 0 {
		def["timeout"] = r.timeout
	}
	return def
}

func recordMap(r *dal.Record) map[string]any {
	out := make(map[string]any, r.Len())
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out[k] = v
	}
	return out
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
	compensate := make(map[string]any, len(j.compensate))
	for k, v := range j.compensate {
		compensate[k] = v
	}
	return map[string]any{
		"data":   map[string]any{"invoke": invoke, "payload": j.payload, "compensate": compensate},
		"errors": append([]any{}, j.errors...),
	}
}
