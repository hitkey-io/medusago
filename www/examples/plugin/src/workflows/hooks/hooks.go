// Package hooks consumes the workflow hooks of the plugin.
package hooks

import (
	"log/slog"

	customerworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: body
// Register consumes the hooks of the plugin: it runs once, when the
// plugin is loaded.
func Register() {
	customerworkflows.CreateCustomersWorkflow.RegisterHook("customersCreated",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			logger := ctx.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
			logger.Info("Wishlist plugin: customers created")

			return composer.NewStepResponse[any](nil), nil
		}, nil)
}

// endregion: body
