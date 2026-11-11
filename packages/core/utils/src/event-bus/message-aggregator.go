package eventbus

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// MessageAggregator mirrors MessageAggregator: the messages saved by a
// module call, deduplicated by their JSON form.
type MessageAggregator struct {
	hashes   map[string]bool
	messages []Message
}

// NewMessageAggregator creates an aggregator.
func NewMessageAggregator() *MessageAggregator {
	return &MessageAggregator{hashes: map[string]bool{}}
}

// Count mirrors count().
func (a *MessageAggregator) Count() int { return len(a.messages) }

func messageHash(m Message) (string, bool) {
	rec := dal.NewRecord()
	rec.Set("name", m.Name)
	rec.Set("metadata", m.Metadata)
	rec.Set("data", m.Data)
	if m.Options != nil {
		rec.Set("options", m.Options)
	}
	raw, err := dal.MarshalValue(rec)
	if err != nil {
		return "", false
	}
	return string(raw), true
}

// Save mirrors save(): identical messages are kept once.
func (a *MessageAggregator) Save(messages ...Message) {
	for _, m := range messages {
		hash, ok := messageHash(m)
		if !ok {
			a.messages = append(a.messages, m)
			continue
		}
		if a.hashes[hash] {
			continue
		}
		a.hashes[hash] = true
		a.messages = append(a.messages, m)
	}
}

// SaveRawMessageData mirrors saveRawMessageData().
func (a *MessageAggregator) SaveRawMessageData(messages ...RawMessage) {
	composed := make([]Message, len(messages))
	for i, m := range messages {
		composed[i] = ComposeMessage(m)
	}
	a.Save(composed...)
}

// GetMessages mirrors getMessages() without grouping or sorting: the
// "default" group.
func (a *MessageAggregator) GetMessages() []Message {
	return append([]Message{}, a.messages...)
}

// ClearMessages mirrors clearMessages().
func (a *MessageAggregator) ClearMessages() {
	a.messages = nil
	a.hashes = map[string]bool{}
}
