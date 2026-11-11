package modulessdk

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
)

// ModuleEventSink mirrors the event part of a module service built by
// MedusaService: the entity mutation events (interceptEntityMutationEvents)
// and the explicitly built ones are aggregated and emitted by emitEvents_
// with { internal: true }.
type ModuleEventSink struct {
	mu  sync.RWMutex
	bus eventbus.Emitter
}

// SetEventBus sets the event bus module (eventBusModuleService_).
func (s *ModuleEventSink) SetEventBus(bus eventbus.Emitter) {
	s.mu.Lock()
	s.bus = bus
	s.mu.Unlock()
}

// EmitModuleEvents mirrors @EmitEvents: the events are saved into a message
// aggregator (deduplicated, with the caller's eventGroupId) and emitted.
func (s *ModuleEventSink) EmitModuleEvents(ctx context.Context, events []dal.ReadyEvent) {
	s.mu.RLock()
	bus := s.bus
	s.mu.RUnlock()
	mode := dal.EventsModeOf(ctx)
	if bus == nil || mode == dal.DropEventsMode {
		return
	}
	groupID := eventbus.EventGroupID(ctx)
	raw := make([]eventbus.RawMessage, len(events))
	for i, ev := range events {
		raw[i] = eventbus.RawMessage{
			EventName:    ev.EventName,
			Source:       ev.Source,
			Object:       ev.Object,
			Action:       ev.Action,
			Data:         ev.Data,
			EventGroupID: groupID,
		}
	}
	var messages []eventbus.Message
	if shared := eventbus.SharedAggregatorOf(ctx); shared != nil {
		messages = shared.Save(raw, mode != dal.SaveEventsMode, mode == dal.EmitEventsMode)
	} else if mode == dal.EmitEventsMode {
		aggregator := eventbus.NewMessageAggregator()
		aggregator.SaveRawMessageData(raw...)
		messages = aggregator.GetMessages()
	}
	if len(messages) == 0 {
		return
	}
	for i := range messages {
		options := dal.NewRecord()
		options.Set("internal", true)
		messages[i].Options = options
	}
	if err := bus.Emit(ctx, messages); err != nil {
		slog.Error(fmt.Sprintf("Failed to emit the module events: %v", err))
	}
}

// ModuleEventBuilderFactory mirrors moduleEventBuilderFactory: the builder
// of one aggregated event { id } (one id) or { id: [...] } (several) for the
// given entities; nil when there are none.
func ModuleEventBuilderFactory(source, action, object, eventName string) func(ids ...any) *dal.ReadyEvent {
	if eventName == "" {
		eventName = eventbus.BuildModuleResourceEventName(source, object, action)
	}
	return func(ids ...any) *dal.ReadyEvent {
		if len(ids) == 0 {
			return nil
		}
		data := dal.NewRecord()
		if len(ids) == 1 {
			data.Set("id", ids[0])
		} else {
			data.Set("id", append([]any{}, ids...))
		}
		return &dal.ReadyEvent{EventName: eventName, Source: source, Object: object, Action: action, Data: data}
	}
}

// SaveEvent saves an explicitly built event (aggregatedEvents /
// eventBuilders) for the current call: into the aggregator of the running
// transaction when db is one, otherwise it is emitted at once (the call's
// @EmitEvents boundary ends with it).
func SaveEvent(ctx context.Context, db dal.Querier, sink dal.EventSink, ev *dal.ReadyEvent) {
	if ev == nil {
		return
	}
	if dal.RecordEvent(db, *ev) {
		return
	}
	if sink != nil {
		sink.EmitModuleEvents(ctx, []dal.ReadyEvent{*ev})
	}
}
