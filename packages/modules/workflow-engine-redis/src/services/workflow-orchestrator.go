package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/oklog/ulid/v2"
	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// WorkflowOrchestrator mirrors the Redis parts of WorkflowOrchestratorService:
// notifications of runs with async steps are published on
// orchestrator:<workflowId> ({ instanceId, data: NotifyOptions }) and the
// ones of other instances are delivered to the local subscribers. The
// responses of waiting steps resume the stored transaction in the instance
// that receives them (composer.DurableEngine, implemented by the storage).
type WorkflowOrchestrator struct {
	instanceID string
	publisher  *ioredis.Client
	subscriber *ioredis.Client
	logger     *slog.Logger
	pubsub     *redis.PubSub
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewWorkflowOrchestrator builds the orchestrator.
func NewWorkflowOrchestrator(publisher, subscriber *ioredis.Client, logger *slog.Logger) *WorkflowOrchestrator {
	return &WorkflowOrchestrator{
		instanceID: ulid.Make().String(),
		publisher:  publisher, subscriber: subscriber, logger: logger,
	}
}

// channelName mirrors getChannelName.
func channelName(workflowID string) string { return "orchestrator:" + workflowID }

// Start subscribes to the notifications of every workflow.
func (o *WorkflowOrchestrator) Start(ctx context.Context) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	o.cancel = cancel
	o.pubsub = o.subscriber.PSubscribe(ctx, channelName("*"))
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
				o.onMessage(msg)
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

func (o *WorkflowOrchestrator) onMessage(msg *redis.Message) {
	if !strings.HasPrefix(msg.Channel, "orchestrator:") {
		return
	}
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
