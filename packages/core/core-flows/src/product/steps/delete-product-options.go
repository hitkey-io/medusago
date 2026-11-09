package steps

import (
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteProductOptionsStep mirrors deleteProductOptionsStep.
var DeleteProductOptionsStep = composer.CreateStep("delete-product-options",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if _, err := ProductService(ctx.Container).SoftDeleteProductOptions(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := ProductService(ctx.Container).Internal("ProductOption").Restore(ctx.Context, ids)
		return err
	})
