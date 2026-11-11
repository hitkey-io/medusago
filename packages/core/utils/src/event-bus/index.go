// Package eventbus mirrors @medusajs/utils/src/event-bus: the subscriber
// types shared by the event bus modules, the helpers of
// AbstractEventBusModuleService, message composition, the message aggregator
// of @EmitEvents and the event names built from entity names.
package eventbus

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Event mirrors the published event body: { name, data, metadata }.
type Event struct {
	Name     string
	Data     any
	Metadata *dal.Record
}

// Subscriber handles a published event.
type Subscriber func(ctx context.Context, e Event) error

// Subscribable is the subscribe part of IEventBusModuleService.
type Subscribable interface {
	Subscribe(event string, sub Subscriber)
}

// EventPriority mirrors EventPriority.
var EventPriority = struct {
	CRITICAL, HIGH, DEFAULT, LOW, LOWEST int
}{CRITICAL: 10, HIGH: 50, DEFAULT: 100, LOW: 500, LOWEST: 1 << 21}

// WithCreatedAtMetadata mirrors withCreatedAtMetadata.
func WithCreatedAtMetadata(m *dal.Record) *dal.Record { return withDate(m, "created_at") }

// WithPublishedAtMetadata mirrors withPublishedAtMetadata.
func WithPublishedAtMetadata(m *dal.Record) *dal.Record { return withDate(m, "published_at") }

func withDate(m *dal.Record, key string) *dal.Record {
	out := dal.NewRecord()
	if m != nil {
		for _, k := range m.Keys() {
			v, _ := m.Get(k)
			out.Set(k, v)
		}
	}
	out.Set(key, time.Now())
	return out
}

// ParseEventMetadataDates mirrors parseEventMetadataDates: created_at and
// published_at back to dates after JSON serialization.
func ParseEventMetadataDates(m *dal.Record) *dal.Record {
	if m == nil {
		return nil
	}
	out := dal.NewRecord()
	for _, k := range m.Keys() {
		v, _ := m.Get(k)
		out.Set(k, v)
	}
	for _, k := range []string{"created_at", "published_at"} {
		if v, ok := m.Get(k); ok && v != nil {
			if s, ok := v.(string); ok {
				if t, ok := common.ParseJSDate(s); ok {
					out.Set(k, t)
				}
			}
		}
	}
	return out
}

// WorkerMode mirrors the module declaration's worker_mode
// (projectConfig.workerMode, MEDUSA_WORKER_MODE): shared, worker or server.
func WorkerMode() string {
	if v := os.Getenv("MEDUSA_WORKER_MODE"); v != "" {
		return v
	}
	return "shared"
}

// IsWorkerMode mirrors isWorkerMode (worker_mode !== "server").
func IsWorkerMode() bool { return WorkerMode() != "server" }

// Message mirrors EventBusTypes.Message: { name, metadata, data, options }.
type Message struct {
	Name     string
	Data     any
	Options  *dal.Record
	Metadata *dal.Record
}

// Emitter is the emit part of IEventBusModuleService.
type Emitter interface {
	Emit(ctx context.Context, messages []Message) error
}

type eventGroupKey struct{}

// WithEventGroupID returns a context carrying the event group of a workflow
// run: what the workflow's medusaContext gives the module services called
// from its steps (sharedContext.eventGroupId).
func WithEventGroupID(ctx context.Context, eventGroupID string) context.Context {
	return context.WithValue(ctx, eventGroupKey{}, eventGroupID)
}

// EventGroupID returns the event group of the context ("" when none).
func EventGroupID(ctx context.Context) string {
	v, _ := ctx.Value(eventGroupKey{}).(string)
	return v
}

// SharedAggregator mirrors the message aggregator a workflow run's
// medusaContext holds: @EmitEvents injects it into the shared context on the
// first call, then every module call of the run saves into it and each
// @EmitEvents call emits (and clears) what it holds.
type SharedAggregator struct {
	mu     sync.Mutex
	agg    *MessageAggregator
	active bool
}

// NewSharedAggregator creates the aggregator of a workflow run.
func NewSharedAggregator() *SharedAggregator {
	return &SharedAggregator{agg: NewMessageAggregator()}
}

type sharedAggregatorKey struct{}

// WithSharedAggregator returns a context carrying the run's aggregator.
func WithSharedAggregator(ctx context.Context, a *SharedAggregator) context.Context {
	return context.WithValue(ctx, sharedAggregatorKey{}, a)
}

// SharedAggregatorOf returns the run's aggregator (nil outside workflows).
func SharedAggregatorOf(ctx context.Context) *SharedAggregator {
	a, _ := ctx.Value(sharedAggregatorKey{}).(*SharedAggregator)
	return a
}

// Save saves messages; activate injects the aggregator first (an
// @EmitEvents call), otherwise the messages are kept only when it already
// exists. When emit is set, the messages held are returned and cleared.
func (s *SharedAggregator) Save(messages []RawMessage, activate, emit bool) []Message {
	s.mu.Lock()
	defer s.mu.Unlock()
	if activate {
		s.active = true
	}
	if !s.active {
		return nil
	}
	s.agg.SaveRawMessageData(messages...)
	if !emit {
		return nil
	}
	out := s.agg.GetMessages()
	s.agg.ClearMessages()
	return out
}
