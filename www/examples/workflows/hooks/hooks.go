// Package hooks holds the examples of the Workflow Hooks chapter.
package hooks

import (
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: consume
// RegisterProductsCreatedHook consumes the productsCreated hook of
// createProductsWorkflow. Call it before the application starts, for
// example from main.go.
func RegisterProductsCreatedHook() {
	workflows.CreateProductsWorkflow.RegisterHook("productsCreated",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			// the hook's input is a record with the created products
			input := data.(*dal.Record)
			products, _ := input.Get("products")

			logger := ctx.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
			logger.Info("Products created", "products", len(products.([]*dal.Record)))

			return composer.NewStepResponse[any](nil), nil
		},
		nil)
}

// endregion: consume

// region: compensation
// RegisterProductsCreatedHookWithCompensation consumes the hook with a
// compensation function, the third argument of RegisterHook.
func RegisterProductsCreatedHookWithCompensation() {
	workflows.CreateProductsWorkflow.RegisterHook("productsCreated",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			// TODO perform an action

			ids := []string{"id_1"}
			return composer.NewStepResponse[any](nil, ids), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			ids, _ := data.([]string)
			_ = ids

			// TODO undo the performed action
			return nil
		})
}

// endregion: compensation

// region: additional-data
// RegisterProductsCreatedHookWithAdditionalData reads the additional data
// the hook's input holds.
func RegisterProductsCreatedHookWithAdditionalData() {
	workflows.CreateProductsWorkflow.RegisterHook("productsCreated",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			input := data.(*dal.Record)

			// additional_data is nil if none was passed
			additionalData, _ := input.Get("additional_data")
			if additional, ok := additionalData.(*dal.Record); ok && additional != nil {
				customField, _ := additional.Get("custom_field")
				_ = customField
			}

			return composer.NewStepResponse[any](nil), nil
		},
		nil)
}

// endregion: additional-data

// region: return-value
// RegisterPricingContextHook consumes a hook that uses the value its
// handler returns: setPricingContext of createAndCompleteReturnOrderWorkflow
// merges it into the pricing context.
func RegisterPricingContextHook() {
	orderworkflows.CreateAndCompleteReturnOrderWorkflow.RegisterHook("setPricingContext",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			pricingContext := dal.NewRecord()
			pricingContext.Set("customer_group_id", "cusgroup_123")

			return composer.NewStepResponse[any](pricingContext), nil
		},
		nil)
}

// endregion: return-value
