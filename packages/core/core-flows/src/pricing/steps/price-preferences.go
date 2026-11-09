package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreatePricePreferencesStep mirrors createPricePreferencesStep.
var CreatePricePreferencesStep = composer.CreateStep("create-price-preferences",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := pricing(ctx.Container).CreatePricePreferences(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(created))
		for i, c := range created {
			ids[i], _ = c.Get("id")
		}
		return composer.NewStepResponse(created, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).Internal("PricePreference").Delete(ctx.Context, ids)
		return err
	})

// UpdatePricePreferencesInput mirrors UpdatePricePreferencesWorkflowInput.
type UpdatePricePreferencesInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdatePricePreferencesStep mirrors updatePricePreferencesStep.
var UpdatePricePreferencesStep = composer.CreateStep("update-price-preferences",
	func(ctx composer.StepExecutionContext, in UpdatePricePreferencesInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := pricing(ctx.Container)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "PricePreferences", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdatePricePreferences(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).UpsertPricePreferences(ctx.Context, prev)
		return err
	})
