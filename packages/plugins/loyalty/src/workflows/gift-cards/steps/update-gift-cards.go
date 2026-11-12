package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateGiftCardsStep mirrors updateGiftCardsStep ("update-gift-cards"):
// the gift cards are updated; the compensation restores the selected
// fields of the gift cards as they were.
var UpdateGiftCardsStep = composer.CreateStep("update-gift-cards",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		module := loyaltyModule(ctx.Container)
		objs := make([]common.ObjectLike, len(data))
		for i, d := range data {
			objs[i] = d
		}
		selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{}, "")
		filters := dal.NewRecord()
		filters.Set("id", ids(data))
		before, err := module.ListBySuffix(ctx.Context, "GiftCards", filters, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := module.Call(ctx.Context, "updateGiftCards", []any{toAny(data)})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated := asRecords(out)
		if updated == nil {
			updated = []*dal.Record{}
		}
		return composer.NewStepResponse(updated, any(map[string]any{"dataBeforeUpdate": before, "selects": selects, "relations": relations})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		in, _ := data.(map[string]any)
		if in == nil {
			return nil
		}
		before, _ := in["dataBeforeUpdate"].([]*dal.Record)
		selects, _ := in["selects"].([]string)
		revert := make([]any, 0, len(before))
		for _, r := range before {
			revert = append(revert, convertItemResponseToUpdateRequest(r, selects))
		}
		if len(revert) == 0 {
			return nil
		}
		_, err := loyaltyModule(ctx.Container).Call(ctx.Context, "updateGiftCards", []any{revert})
		return err
	})

// convertItemResponseToUpdateRequest mirrors
// convertItemResponseToUpdateRequest for the scalar fields of a gift card:
// the id and the selected fields.
func convertItemResponseToUpdateRequest(item *dal.Record, selects []string) *dal.Record {
	out := dal.NewRecord()
	out.Set("id", get(item, "id"))
	for _, k := range selects {
		if k == "id" {
			continue
		}
		if v, ok := item.Get(k); ok {
			out.Set(k, v)
		}
	}
	return out
}
