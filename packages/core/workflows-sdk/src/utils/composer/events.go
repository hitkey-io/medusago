package composer

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// WorkflowEvent mirrors the NotifyOptions delivered to workflow engine
// subscribers (onBegin, onStepBegin, onStepSuccess, onStepFailure,
// onStepAwaiting, onCompensateBegin, onCompensateStepSuccess,
// onCompensateStepFailure, onFinish).
type WorkflowEvent struct {
	EventType     string
	WorkflowID    string
	TransactionID string
	// Step is the TransactionStep snapshot (nil when absent).
	Step *dal.Record
	// Response, Result and Errors are set only when their Has* flag is.
	Response    any
	HasResponse bool
	Result      any
	HasResult   bool
	Errors      []any
	HasErrors   bool
	// IsFlowAsync and State mirror NotifyOptions.isFlowAsync / state.
	IsFlowAsync bool
	State       string
}

// WorkflowEventPublisher is implemented by a workflow engine module that
// shares the notifications with other instances (the Redis engine
// publishes those of runs with async steps).
type WorkflowEventPublisher interface {
	PublishWorkflowEvent(ev WorkflowEvent)
}

func eventPublisher(c Container) WorkflowEventPublisher {
	if c == nil {
		return nil
	}
	p, _ := c.Resolve(utilsmodulessdk.WORKFLOW_ENGINE).(WorkflowEventPublisher)
	return p
}

// DeliverWorkflowEvent delivers a notification received from another
// instance to the subscribers of this one.
func DeliverWorkflowEvent(ev WorkflowEvent) { notifyEvent(ev) }

// HasSubscribers reports whether this process has subscribers of the
// workflow.
func HasSubscribers(workflowID string) bool {
	subsMu.Lock()
	defer subsMu.Unlock()
	return len(subs[workflowID]) > 0
}

// NotifyOptions mirrors the NotifyOptions published by the Redis engine:
// { eventType, isFlowAsync, workflowId, transactionId, step, response,
// result, errors, state }.
func (ev WorkflowEvent) NotifyOptions() *dal.Record {
	r := dal.NewRecord()
	r.Set("eventType", ev.EventType)
	r.Set("isFlowAsync", ev.IsFlowAsync)
	r.Set("workflowId", ev.WorkflowID)
	r.Set("transactionId", ev.TransactionID)
	if ev.Step != nil {
		r.Set("step", ev.Step)
	}
	if ev.HasResponse {
		r.Set("response", ev.Response)
	}
	if ev.HasResult {
		r.Set("result", ev.Result)
	}
	if ev.HasErrors {
		r.Set("errors", ev.Errors)
	}
	if ev.State != "" {
		r.Set("state", ev.State)
	}
	return r
}

// WorkflowEventFromNotifyOptions reads NotifyOptions (JSON decoded).
func WorkflowEventFromNotifyOptions(r *dal.Record) WorkflowEvent {
	var ev WorkflowEvent
	str := func(k string) string {
		v, _ := r.Get(k)
		s, _ := v.(string)
		return s
	}
	ev.EventType = str("eventType")
	ev.WorkflowID = str("workflowId")
	ev.TransactionID = str("transactionId")
	ev.State = str("state")
	if v, ok := r.Get("isFlowAsync"); ok {
		ev.IsFlowAsync, _ = v.(bool)
	}
	if v, ok := r.Get("step"); ok {
		ev.Step, _ = v.(*dal.Record)
	}
	if v, ok := r.Get("response"); ok {
		ev.Response, ev.HasResponse = v, true
	}
	if v, ok := r.Get("result"); ok {
		ev.Result, ev.HasResult = v, true
	}
	if v, ok := r.Get("errors"); ok {
		ev.Errors, _ = v.([]any)
		ev.HasErrors = true
	}
	return ev
}

// Subscriber receives workflow events.
type Subscriber func(WorkflowEvent)

// anySubscriber mirrors AnySubscriber (all transactions of a workflow).
const anySubscriber = "__any__"

type subscription struct {
	id string
	fn Subscriber
}

var (
	subsMu sync.Mutex
	// subs[workflowID][transactionID or anySubscriber]
	subs   = map[string]map[string][]subscription{}
	evQ    = make(chan WorkflowEvent, 1024)
	evOnce sync.Once
)

// Subscribe mirrors workflowEngine.subscribe({ workflowId, transactionId?,
// subscriber, subscriberId }).
func Subscribe(workflowID, transactionID, subscriberID string, fn Subscriber) {
	key := transactionID
	if key == "" {
		key = anySubscriber
	}
	subsMu.Lock()
	defer subsMu.Unlock()
	byTx := subs[workflowID]
	if byTx == nil {
		byTx = map[string][]subscription{}
		subs[workflowID] = byTx
	}
	for _, s := range byTx[key] {
		if s.id == subscriberID {
			return
		}
	}
	byTx[key] = append(byTx[key], subscription{id: subscriberID, fn: fn})
}

// Unsubscribe mirrors workflowEngine.unsubscribe({ workflowId,
// transactionId?, subscriberOrId }).
func Unsubscribe(workflowID, transactionID, subscriberID string) {
	key := transactionID
	if key == "" {
		key = anySubscriber
	}
	subsMu.Lock()
	defer subsMu.Unlock()
	byTx := subs[workflowID]
	if byTx == nil {
		return
	}
	list := byTx[key]
	kept := list[:0]
	for _, s := range list {
		if s.id != subscriberID {
			kept = append(kept, s)
		}
	}
	if len(kept) == 0 {
		delete(byTx, key)
	} else {
		byTx[key] = kept
	}
}

// notifyEvent mirrors notify(): subscribers are processed asynchronously
// (setImmediate upstream), in order.
func notifyEvent(ev WorkflowEvent) {
	evOnce.Do(func() {
		go func() {
			for ev := range evQ {
				deliverEvent(ev)
			}
		}()
	})
	evQ <- ev
}

func deliverEvent(ev WorkflowEvent) {
	subsMu.Lock()
	byTx := subs[ev.WorkflowID]
	var handlers []subscription
	if byTx != nil {
		if ev.TransactionID != "" {
			handlers = append(handlers, byTx[ev.TransactionID]...)
			if ev.EventType == "onFinish" {
				delete(byTx, ev.TransactionID)
			}
		}
		handlers = append(handlers, byTx[anySubscriber]...)
	}
	subsMu.Unlock()
	for _, h := range handlers {
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error(fmt.Sprintf("Subscriber error: %v", r))
				}
			}()
			h.fn(ev)
		}()
	}
}

// stepSnapshot mirrors a serialized TransactionStep (caller holds j.mu).
func (j *journal) stepSnapshot(r *stepRecord) *dal.Record {
	def := dal.NewRecord()
	def.Set("action", r.action)
	def.Set("uuid", r.uuid)
	if r.async {
		def.Set("async", true)
		def.Set("compensateAsync", r.compensateAsync)
	}
	def.Set("noCompensation", r.noCompensation)
	def.Set("store", r.store)
	state := func(s, st string) *dal.Record {
		o := dal.NewRecord()
		o.Set("state", s)
		o.Set("status", st)
		return o
	}
	out := dal.NewRecord()
	out.Set("stepFailed", r.stepFailed)
	out.Set("id", r.id)
	out.Set("uuid", r.uuid)
	out.Set("depth", r.depth)
	out.Set("definition", def)
	out.Set("saveResponse", true)
	out.Set("invoke", state(r.invokeState, r.invokeStatus))
	out.Set("compensate", state(r.compState, r.compStatus))
	out.Set("attempts", r.attempts)
	out.Set("failures", r.failures)
	out.Set("lastAttempt", r.lastAttempt)
	out.Set("next", append([]string{}, r.next...))
	out.Set("_v", j.version)
	out.Set("startedAt", r.startedAt)
	return out
}

// emit sends an event when the run notifies (runs started by the workflow
// engine); the caller holds j.mu.
func (j *journal) emit(eventType string, r *stepRecord, fill func(*WorkflowEvent)) {
	if j.events == nil {
		return
	}
	ev := WorkflowEvent{EventType: eventType}
	if r != nil {
		ev.Step = j.stepSnapshot(r)
	}
	if fill != nil {
		fill(&ev)
	}
	j.events(ev)
}

// errorsFor mirrors transaction.getErrors(handlerType) filtered by action
// (caller holds j.mu).
func (j *journal) errorsFor(action, handlerType string) []any {
	out := []any{}
	for _, e := range j.errors {
		m, _ := e.(map[string]any)
		if m["action"] == action && m["handlerType"] == handlerType {
			out = append(out, e)
		}
	}
	return out
}
