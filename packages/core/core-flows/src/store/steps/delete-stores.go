package steps

import (
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteStoresStep mirrors deleteStoresStep: the stores are soft deleted and
// restored on compensation.
var DeleteStoresStep = composer.CreateStep("delete-stores",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if _, err := StoreService(ctx.Container).Call(ctx.Context, "softDeleteStores", []any{ids}); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := StoreService(ctx.Container).Call(ctx.Context, "restoreStores", []any{ids})
		return err
	})
