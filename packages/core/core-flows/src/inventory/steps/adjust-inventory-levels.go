package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type adjuster interface {
	AdjustInventory(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

func adjustInput(list []*dal.Record, negate bool) []*dal.Record {
	out := make([]*dal.Record, len(list))
	for i, item := range list {
		r := dal.NewRecord()
		r.Set("inventoryItemId", get(item, "inventory_item_id"))
		r.Set("locationId", get(item, "location_id"))
		adj := get(item, "adjustment")
		if negate {
			adj = totals.Mult(adj, -1)
		}
		r.Set("adjustment", adj)
		out[i] = r
	}
	return out
}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// AdjustInventoryLevelsStep mirrors adjustInventoryLevelsStep (locks not
// ported).
var AdjustInventoryLevelsStep = composer.CreateStep("adjust-inventory-levels-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(input) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := ctx.Container.Resolve(utilsmodulessdk.INVENTORY).(adjuster)
		levels, err := svc.AdjustInventory(ctx.Context, adjustInput(input, false))
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(levels, any(input)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		input, _ := data.([]*dal.Record)
		if len(input) == 0 {
			return nil
		}
		_, err := ctx.Container.Resolve(utilsmodulessdk.INVENTORY).(adjuster).AdjustInventory(ctx.Context, adjustInput(input, true))
		return err
	})
