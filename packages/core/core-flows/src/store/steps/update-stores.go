// Package steps mirrors @medusajs/core-flows/src/store/steps.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/store/src/services"
)

// StoreService resolves the store module service.
func StoreService(c composer.Container) *services.StoreModuleService {
	s, _ := c.Resolve(utilsmodulessdk.STORE).(*services.StoreModuleService)
	return s
}

// UpdateStoresStepInput mirrors UpdateStoresStepInput.
type UpdateStoresStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateStoresStep mirrors updateStoresStep.
var UpdateStoresStep = composer.CreateStep("update-stores",
	func(ctx composer.StepExecutionContext, in UpdateStoresStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := StoreService(ctx.Container)
		selects, relations := common.GetSelectsAndRelationsFromObjectArray([]common.ObjectLike{in.Update}, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "Stores", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateStores(ctx.Context, in.Selector, in.Update)
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
		in := make([]*dal.Record, len(prev))
		for i, r := range prev {
			n := dal.NewRecord()
			for _, k := range r.Keys() {
				v, _ := r.Get(k)
				if k == "metadata" && v == nil {
					continue
				}
				n.Set(k, v)
			}
			in[i] = n
		}
		_, err := StoreService(ctx.Container).UpsertStores(ctx.Context, in)
		return err
	})
