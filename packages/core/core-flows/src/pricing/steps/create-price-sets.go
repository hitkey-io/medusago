// Package steps mirrors @medusajs/core-flows/src/pricing/steps.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/pricing/src/services"
)

func pricing(c composer.Container) *services.PricingModuleService {
	s, _ := c.Resolve(utilsmodulessdk.PRICING).(*services.PricingModuleService)
	return s
}

// CreatePriceSetsStep mirrors createPriceSetsStep.
var CreatePriceSetsStep = composer.CreateStep("create-price-sets",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		sets, err := pricing(ctx.Container).CreatePriceSets(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(sets))
		for i, s := range sets {
			ids[i], _ = s.Get("id")
		}
		return composer.NewStepResponse(sets, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return pricing(ctx.Container).DeletePriceSets(ctx.Context, ids)
	})
