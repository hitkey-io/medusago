// Package eventbus mirrors @medusajs/utils/src/event-bus: message
// composition, the message aggregator of @EmitEvents and the event names
// built from entity names.
package eventbus

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

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
