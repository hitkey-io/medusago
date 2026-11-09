package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdatePriceSetsStepInput mirrors UpdatePriceSetsStepInput: PriceSets
// (batch, when Batch) or Selector/Update.
type UpdatePriceSetsStepInput struct {
	Batch     bool
	PriceSets []*dal.Record
	Selector  *dal.Record
	Update    *dal.Record
}

// UpdatePriceSetsStep mirrors updatePriceSetsStep.
var UpdatePriceSetsStep = composer.CreateStep("update-price-sets",
	func(ctx composer.StepExecutionContext, in UpdatePriceSetsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := pricing(ctx.Container)
		if in.Batch {
			ids := make([]any, 0, len(in.PriceSets))
			for _, p := range in.PriceSets {
				id, _ := p.Get("id")
				if id == nil || id == "" {
					return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Price set id is required when doing a batch update")
				}
				ids = append(ids, id)
			}
			f := dal.NewRecord()
			f.Set("id", ids)
			prev, err := svc.ListBySuffix(ctx.Context, "PriceSets", f, utilsmodulessdk.FindConfig{})
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			sets, err := svc.UpsertPriceSets(ctx.Context, in.PriceSets)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			return composer.NewStepResponse(sets, any(prev)), nil
		}
		if in.Selector == nil || in.Update == nil {
			return composer.NewStepResponse([]*dal.Record{}, any(nil)), nil
		}
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{ObjectFields: []string{"rules"}}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "PriceSets", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		sets, err := svc.UpdatePriceSets(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(sets, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).UpsertPriceSets(ctx.Context, prev)
		return err
	})
