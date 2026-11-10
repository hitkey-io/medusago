package composer

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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
