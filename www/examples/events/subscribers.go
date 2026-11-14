// Package events holds the subscribers, the workflow and the service that
// emit and handle events in the event chapters of the Learn documentation.
// In a project the subscribers live under src/subscribers.
package events

import (
	"context"
	"errors"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
)

// region: order-placed
// orderPlacedHandler returns the subscriber of the order.placed event. It
// runs a workflow that sends a confirmation email.
func orderPlacedHandler(container composer.Container) eventbus.Subscriber {
	return func(ctx context.Context, event eventbus.Event) error {
		logger, _ := container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
		logger.Info("Sending confirmation email...")

		_, err := SendOrderConfirmationWorkflow.Run(ctx, container, SendOrderConfirmationInput{
			ID: eventID(event),
		})
		return err
	}
}

// endregion: order-placed

// region: register
// RegisterSubscribers subscribes the handlers to their events. Pass it to
// cli.Options.OnLoad.
func RegisterSubscribers(ctx context.Context, app *loaders.MedusaApp) error {
	bus, ok := app.Container.Resolve(utilsmodulessdk.EVENT_BUS).(eventbus.Subscribable)
	if !ok {
		return errors.New("the event module doesn't support subscribers")
	}

	bus.Subscribe("order.placed", orderPlacedHandler(app.Container))
	bus.Subscribe("product.created", productCreatedHandler(app.Container))

	return nil
}

// endregion: register
