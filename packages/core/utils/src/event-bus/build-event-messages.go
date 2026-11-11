package eventbus

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RawMessage mirrors EventBusTypes.RawMessageFormat.
type RawMessage struct {
	EventName string
	Source    string
	Object    string
	Action    string
	Data      any
	// EventGroupID is context.eventGroupId of the caller.
	EventGroupID string
	Options      *dal.Record
}

// ComposeMessage mirrors composeMessage: the message with its metadata
// { source, object, action, eventGroupId? }.
func ComposeMessage(m RawMessage) Message {
	metadata := dal.NewRecord()
	metadata.Set("source", m.Source)
	metadata.Set("object", m.Object)
	metadata.Set("action", m.Action)
	if m.EventGroupID != "" {
		metadata.Set("eventGroupId", m.EventGroupID)
	}
	return Message{Name: m.EventName, Metadata: metadata, Data: m.Data, Options: m.Options}
}
