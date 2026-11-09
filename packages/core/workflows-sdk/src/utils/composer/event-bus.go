package composer

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Message mirrors EventBusTypes.Message.
type Message struct {
	Name     string
	Data     any
	Options  *dal.Record
	Metadata *dal.Record
}

// EventBus is the part of IEventBusModuleService the workflows use.
type EventBus interface {
	Emit(ctx context.Context, messages []Message) error
	ReleaseGroupedEvents(ctx context.Context, eventGroupID string) error
	// ClearGroupedEvents clears the group (eventNames nil = all events).
	ClearGroupedEvents(ctx context.Context, eventGroupID string, eventNames []string) error
}
