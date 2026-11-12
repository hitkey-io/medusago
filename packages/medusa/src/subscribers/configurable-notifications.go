package subscribers

import (
	"context"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type notificationHandlerConfig struct {
	event      string
	template   string
	channel    string
	to         string
	resourceID string
	data       [][2]string // ordered { key: path }
}

// handlerConfig mirrors the handlerConfig of configurable-notifications.ts.
var handlerConfig = []notificationHandlerConfig{
	{
		event:      "order.created",
		template:   "order-created-template",
		channel:    "email",
		to:         "order.email",
		resourceID: "order.id",
		data:       [][2]string{{"order_id", "order.id"}},
	},
}

// ConfigurableNotificationsSubscriberID mirrors config.context.subscriberId.
const ConfigurableNotificationsSubscriberID = "configurable-notifications-handler"

// ConfigurableNotificationsEvents mirrors config.event.
func ConfigurableNotificationsEvents() []string {
	out := []string{}
	seen := map[string]bool{}
	for _, h := range handlerConfig {
		if !seen[h.event] {
			seen[h.event] = true
			out = append(out, h.event)
		}
	}
	return out
}

type notificationCreator interface {
	CreateNotifications(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

// ConfigurableNotificationsHandler mirrors configurableNotifications: every
// handler of the event creates a notification whose values are picked from
// the event payload (undefined values are left out); failures are logged,
// not returned.
func ConfigurableNotificationsHandler(ctx context.Context, c composer.Container, eventName string, payload any) error {
	logger, _ := c.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	svc, _ := c.Resolve(utilsmodulessdk.NOTIFICATION).(notificationCreator)
	for _, h := range handlerConfig {
		if h.event != eventName {
			continue
		}
		n := dal.NewRecord()
		n.Set("template", h.template)
		n.Set("channel", h.channel)
		if v, ok := dal.PickValueFromObject(h.to, payload); ok {
			n.Set("to", v)
		}
		n.Set("trigger_type", h.event)
		if v, ok := dal.PickValueFromObject(h.resourceID, payload); ok {
			n.Set("resource_id", v)
		}
		data := dal.NewRecord()
		for _, kv := range h.data {
			if v, ok := dal.PickValueFromObject(kv[1], payload); ok {
				data.Set(kv[0], v)
			}
		}
		n.Set("data", data)
		if svc == nil {
			continue
		}
		// We don't want to fail all handlers, so we catch and log errors only
		if _, err := svc.CreateNotifications(ctx, []*dal.Record{n}); err != nil && logger != nil {
			logger.Error("Failed to send notification for "+eventName, "error", err.Error())
		}
	}
	return nil
}
