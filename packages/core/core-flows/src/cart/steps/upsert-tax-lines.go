package steps

import (
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func withExistingIDs(lines []*dal.Record, existing []*dal.Record, ownerKey, ownerFrom string) []*dal.Record {
	out := normalizeTaxLines(lines, ownerKey, ownerFrom)
	for i, l := range out {
		var id any
		for _, e := range existing {
			if Get(e, ownerKey) == Get(lines[i], ownerFrom) {
				id = Get(e, "id")
				break
			}
		}
		n := dal.NewRecord()
		if id != nil {
			n.Set("id", id)
		}
		for _, k := range l.Keys() {
			v, _ := l.Get(k)
			n.Set(k, v)
		}
		out[i] = n
	}
	return out
}

// UpsertTaxLinesForItemsStep mirrors upsertTaxLinesForItemsStep.
var UpsertTaxLinesForItemsStep = composer.CreateStep("upsert-tax-lines-for-items",
	func(ctx composer.StepExecutionContext, in SetTaxLinesInput) (composer.StepResponse[any], error) {
		svc := CartOf(ctx.Container)
		var shipIDs, itemIDs []any
		for _, l := range in.ShippingTaxLines {
			shipIDs = append(shipIDs, Get(l, "shipping_line_id"))
		}
		for _, l := range in.ItemTaxLines {
			itemIDs = append(itemIDs, Get(l, "line_item_id"))
		}
		existingShip, err := listTaxLines(ctx.Context, svc, "ShippingMethodTaxLines", "shipping_method_id", shipIDs)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		existingItems, err := listTaxLines(ctx.Context, svc, "LineItemTaxLines", "item_id", itemIDs)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		items := withExistingIDs(in.ItemTaxLines, existingItems, "item_id", "line_item_id")
		ships := withExistingIDs(in.ShippingTaxLines, existingShip, "shipping_method_id", "shipping_line_id")
		if len(items) > 0 {
			if _, err := svc.Internal("LineItemTaxLine").Upsert(ctx.Context, items); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		if len(ships) > 0 {
			if _, err := svc.Internal("ShippingMethodTaxLine").Upsert(ctx.Context, ships); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(taxLinesRevert{Str(in.Cart, "id"), existingItems, existingShip})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, ok := data.(taxLinesRevert)
		if !ok {
			return nil
		}
		svc := CartOf(ctx.Container)
		restore := func(lines []*dal.Record, owner string) []*dal.Record {
			out := make([]*dal.Record, len(lines))
			for i, l := range lines {
				x := dal.NewRecord()
				for _, k := range []string{"description", "tax_rate_id", "code", "rate", "provider_id", owner, "metadata", "data"} {
					x.Set(k, Get(l, k))
				}
				out[i] = x
			}
			return out
		}
		if r.existingItemLines != nil {
			if _, err := svc.Internal("LineItemTaxLine").Upsert(ctx.Context, restore(r.existingItemLines, "item_id")); err != nil {
				return err
			}
		}
		_, err := svc.Internal("ShippingMethodTaxLine").Upsert(ctx.Context, restore(r.existingShippingMethods, "shipping_method_id"))
		return err
	})
