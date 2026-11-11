package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// stepResponseChannel is where a Go instance asks the instance holding a
// waiting async step to complete it (setStepSuccess / setStepFailure
// received by another instance). Upstream resumes the stored checkpoint on
// any instance instead; Go runs keep their state in the process that runs
// them, so the response is relayed to it.
const stepResponseChannel = "medusago:workflows:step-response"

// relayTimeout bounds the wait for the owner's acknowledgement.
const relayTimeout = 2 * time.Second

// WorkflowOrchestrator mirrors the Redis parts of WorkflowOrchestratorService:
// notifications of runs with async steps are published on
// orchestrator:<workflowId> ({ instanceId, data: NotifyOptions }) and the
// ones of other instances are delivered to the local subscribers.
type WorkflowOrchestrator struct {
	*composer.LocalStepCoordinator
	instanceID string
	publisher  *ioredis.Client
	subscriber *ioredis.Client
	logger     *slog.Logger
	pubsub     *redis.PubSub
	cancel     context.CancelFunc
	done       chan struct{}

	mu      sync.Mutex
	pending map[string]chan bool
}

// NewWorkflowOrchestrator builds the orchestrator.
func NewWorkflowOrchestrator(publisher, subscriber *ioredis.Client, logger *slog.Logger) *WorkflowOrchestrator {
	return &WorkflowOrchestrator{
		LocalStepCoordinator: composer.NewLocalStepCoordinator(),
		instanceID:           ulid.Make().String(),
		publisher:            publisher, subscriber: subscriber, logger: logger,
		pending: map[string]chan bool{},
	}
}

// channelName mirrors getChannelName.
func channelName(workflowID string) string { return "orchestrator:" + workflowID }

func (o *WorkflowOrchestrator) replyChannel() string {
	return stepResponseChannel + ":" + o.instanceID
}

// Start subscribes to the notifications of every workflow and to the step
// responses.
func (o *WorkflowOrchestrator) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	o.cancel = cancel
	o.pubsub = o.subscriber.PSubscribe(ctx, channelName("*"))
	if err := o.pubsub.Subscribe(ctx, stepResponseChannel, o.replyChannel()); err != nil {
		return err
	}
	if _, err := o.pubsub.Receive(ctx); err != nil {
		return err
	}
	o.done = make(chan struct{})
	ch := o.pubsub.Channel()
	go func() {
		defer close(o.done)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				o.onMessage(ctx, msg)
			}
		}
	}()
	return nil
}

// Stop closes the subscriptions.
func (o *WorkflowOrchestrator) Stop() {
	if o.cancel != nil {
		o.cancel()
	}
	if o.pubsub != nil {
		_ = o.pubsub.Close()
	}
	if o.done != nil {
		<-o.done
	}
}

type stepResponseMessage struct {
	RequestID  string                  `json:"requestId"`
	InstanceID string                  `json:"instanceId"`
	Key        composer.IdempotencyKey `json:"key"`
	Response   json.RawMessage         `json:"response,omitempty"`
	Error      *string                 `json:"error,omitempty"`
	Delivered  bool                    `json:"delivered,omitempty"`
}

func (o *WorkflowOrchestrator) onMessage(ctx context.Context, msg *redis.Message) {
	switch {
	case msg.Channel == stepResponseChannel:
		var m stepResponseMessage
		if json.Unmarshal([]byte(msg.Payload), &m) != nil || m.InstanceID == o.instanceID || !o.IsWaiting(m.Key) {
			return
		}
		r := composer.StepResult{}
		if m.Error != nil {
			r.Err = errors.New(*m.Error)
		} else if len(m.Response) > 0 {
			v, err := dal.DecodeOrderedJSON(m.Response)
			if err == nil {
				r.Output = v
			}
		}
		if o.DeliverLocal(m.Key, r) {
			b, _ := json.Marshal(stepResponseMessage{RequestID: m.RequestID, InstanceID: o.instanceID, Key: m.Key, Delivered: true})
			_ = o.publisher.Publish(ctx, stepResponseChannel+":"+m.InstanceID, string(b)).Err()
		}
	case msg.Channel == o.replyChannel():
		var m stepResponseMessage
		if json.Unmarshal([]byte(msg.Payload), &m) != nil {
			return
		}
		o.mu.Lock()
		ch := o.pending[m.RequestID]
		delete(o.pending, m.RequestID)
		o.mu.Unlock()
		if ch != nil {
			ch <- m.Delivered
		}
	case strings.HasPrefix(msg.Channel, "orchestrator:"):
		workflowID := strings.SplitN(msg.Channel, ":", 2)[1]
		if !composer.HasSubscribers(workflowID) {
			return
		}
		v, err := dal.DecodeOrderedJSON([]byte(msg.Payload))
		if err != nil {
			o.logger.Error(fmt.Sprintf("Failed to process Redis message: %v", err))
			return
		}
		rec, _ := v.(*dal.Record)
		if rec == nil {
			return
		}
		if id, _ := rec.Get("instanceId"); id == o.instanceID {
			return
		}
		data, _ := rec.Get("data")
		if d, ok := data.(*dal.Record); ok {
			composer.DeliverWorkflowEvent(composer.WorkflowEventFromNotifyOptions(d))
		}
	}
}

// PublishWorkflowEvent mirrors notify(): the notifications of flows with
// async steps are published for the other instances.
func (o *WorkflowOrchestrator) PublishWorkflowEvent(ev composer.WorkflowEvent) {
	if !ev.IsFlowAsync {
		return
	}
	msg := dal.NewRecord()
	msg.Set("instanceId", o.instanceID)
	msg.Set("data", ev.NotifyOptions())
	b, err := json.Marshal(msg)
	if err != nil {
		o.logger.Error(fmt.Sprintf("Failed to publish to Redis: %v", err))
		return
	}
	go func() {
		if err := o.publisher.Publish(context.Background(), channelName(ev.WorkflowID), string(b)).Err(); err != nil {
			o.logger.Error(fmt.Sprintf("Failed to publish to Redis: %v", err))
		}
	}()
}

// Deliver implements composer.StepCoordinator: a step waiting in this
// process, else in another instance.
func (o *WorkflowOrchestrator) Deliver(ctx context.Context, key composer.IdempotencyKey, r composer.StepResult) error {
	if o.DeliverLocal(key, r) {
		return nil
	}
	if o.pubsub != nil {
		delivered, err := o.relay(ctx, key, r)
		if err != nil {
			return err
		}
		if delivered {
			return nil
		}
	}
	return composer.NotFoundWaiting(key)
}

func (o *WorkflowOrchestrator) relay(ctx context.Context, key composer.IdempotencyKey, r composer.StepResult) (bool, error) {
	m := stepResponseMessage{RequestID: ulid.Make().String(), InstanceID: o.instanceID, Key: key}
	if r.Err != nil {
		msg := r.Err.Error()
		m.Error = &msg
	} else {
		b, err := json.Marshal(r.Output)
		if err != nil {
			return false, err
		}
		m.Response = b
	}
	b, err := json.Marshal(m)
	if err != nil {
		return false, err
	}
	ch := make(chan bool, 1)
	o.mu.Lock()
	o.pending[m.RequestID] = ch
	o.mu.Unlock()
	defer func() {
		o.mu.Lock()
		delete(o.pending, m.RequestID)
		o.mu.Unlock()
	}()
	n, err := o.publisher.Publish(ctx, stepResponseChannel, string(b)).Result()
	if err != nil {
		return false, err
	}
	if n <= 1 {
		// only this instance listens
		return false, nil
	}
	select {
	case ok := <-ch:
		return ok, nil
	case <-time.After(relayTimeout):
		return false, nil
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
