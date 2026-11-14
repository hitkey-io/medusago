package events

import (
	"context"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: product-created
// productCreatedHandler returns the subscriber of the product.created event.
func productCreatedHandler(container composer.Container) eventbus.Subscriber {
	return func(ctx context.Context, event eventbus.Event) error {
		logger, _ := container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)

		logger.Info("The product was created", "event", event.Name, "product_id", eventID(event))
		return nil
	}
}

// endregion: product-created

// region: event-id
// eventID reads the id property of an event's payload.
func eventID(event eventbus.Event) string {
	// The payload of the events Medusa emits is an object, with the id of the
	// resource the event is about.
	data, ok := event.Data.(*dal.Record)
	if !ok || data == nil {
		return ""
	}

	id, _ := data.Get("id")
	s, _ := id.(string)
	return s
}

// endregion: event-id

// region: metadata
// describeEvent reads the details of an event.
func describeEvent(event eventbus.Event) (name string, source any, createdAt any) {
	// event.Metadata is nil when the event has no metadata.
	if event.Metadata != nil {
		source, _ = event.Metadata.Get("source")
		createdAt, _ = event.Metadata.Get("created_at")
	}
	return event.Name, source, createdAt
}

// endregion: metadata
