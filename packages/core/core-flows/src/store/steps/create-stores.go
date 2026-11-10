package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateStoresStep mirrors createStoresStep.
var CreateStoresStep = composer.CreateStep("create-stores",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := StoreService(ctx.Container).CreateStores(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(created))
		for i, s := range created {
			ids[i], _ = s.Get("id")
		}
		return composer.NewStepResponse(created, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := StoreService(ctx.Container).Internal("Store").Delete(ctx.Context, ids)
		return err
	})
