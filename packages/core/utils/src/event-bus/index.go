// Package eventbus mirrors @medusajs/utils/src/event-bus: the subscriber
// types shared by the event bus modules and the helpers of
// AbstractEventBusModuleService.
package eventbus

import (
	"context"
	"os"
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
