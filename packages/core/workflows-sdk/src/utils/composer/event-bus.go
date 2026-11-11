package composer

import (
	"context"

	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
)

// Message mirrors EventBusTypes.Message.
type Message = eventbus.Message

// EventBus is the part of IEventBusModuleService the workflows use.
type EventBus interface {
	Emit(ctx context.Context, messages []Message) error
	ReleaseGroupedEvents(ctx context.Context, eventGroupID string) error
	// ClearGroupedEvents clears the group (eventNames nil = all events).
	ClearGroupedEvents(ctx context.Context, eventGroupID string, eventNames []string) error
}
