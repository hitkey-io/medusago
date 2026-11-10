package steps

import (
	"strings"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// LineItemsInput is { id: cart id, items }.
type LineItemsInput struct {
	ID    string
	Items []*dal.Record
}

func existingVariantItems(ctx composer.StepExecutionContext, cartID string, items []*dal.Record) (map[string][]*dal.Record, error) {
	var ids []any
	for _, i := range items {
		ids = append(ids, Get(i, "variant_id"))
	}
	f := dal.NewRecord()
	f.Set("cart_id", cartID)
	f.Set("variant_id", ids)
	rows, err := CartOf(ctx.Container).ListBySuffix(ctx.Context, "LineItems", f, utilsmodulessdk.FindConfig{Select: LineItemFieldsForMerging})
	if err != nil {
		return nil, err
	}
	out := map[string][]*dal.Record{}
	for _, r := range rows {
		v := Str(r, "variant_id")
		out[v] = append(out[v], r)
	}
	return out, nil
}

func toNumber(v *totals.BN) float64 {
	f, _ := v.Float64()
	return f
}

// GetLineItemPricingQuantitiesStep mirrors getLineItemPricingQuantitiesStep.
var GetLineItemPricingQuantitiesStep = composer.CreateStep("get-line-item-pricing-quantities",
	func(ctx composer.StepExecutionContext, in LineItemsInput) (composer.StepResponse[[]any], error) {
		quantities := make([]any, len(in.Items))
		var withVariant []*dal.Record
		for i, item := range in.Items {
			quantities[i] = toNumber(totals.Convert(Coalesce(Get(item, "quantity"), 1)))
			if Str(item, "variant_id") != "" {
				withVariant = append(withVariant, item)
			}
		}
		if len(withVariant) == 0 {
			return composer.NewStepResponse(quantities), nil
		}
		byVariant, err := existingVariantItems(ctx, in.ID, withVariant)
		if err != nil {
			return composer.StepResponse[[]any]{}, err
		}
		for i, item := range in.Items {
			v := Str(item, "variant_id")
			if v == "" {
				continue
			}
			probe := dal.NewRecord()
			probe.Set("metadata", Get(item, "metadata"))
			probe.Set("unit_price", Get(item, "unit_price"))
			probe.Set("is_custom_price", Has(item, "unit_price"))
			existing := FindMatchingLineItem(byVariant[v], probe)
			if existing == nil {
				continue
			}
			q := totals.Add(0, Get(existing, "quantity"), Coalesce(Get(item, "quantity"), 1))
			existing.Set("quantity", totals.NewBN(q))
			quantities[i] = toNumber(q)
		}
		return composer.NewStepResponse(quantities), nil
	}, nil)

// ValidateLineItemPricesStep mirrors validateLineItemPricesStep.
var ValidateLineItemPricesStep = composer.CreateStep("validate-line-item-prices",
	func(ctx composer.StepExecutionContext, items []*dal.Record) (composer.StepResponse[any], error) {
		var missing []string
		for _, i := range items {
			if !IsPresent(Get(i, "unit_price")) {
				missing = append(missing, Str(i, "title"))
			}
		}
		if len(missing) > 0 {
			return composer.StepResponse[any]{}, invalid("Items %s do not have a price", strings.Join(missing, ", "))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// LineItemActions mirrors GetLineItemActionsStepOutput.
type LineItemActions struct {
	ItemsToCreate []*dal.Record
	ItemsToUpdate []*dal.Record
}

// GetLineItemActionsStep mirrors getLineItemActionsStep.
var GetLineItemActionsStep = composer.CreateStep("get-line-item-actions-step",
	func(ctx composer.StepExecutionContext, in LineItemsInput) (composer.StepResponse[LineItemActions], error) {
		out := LineItemActions{ItemsToCreate: []*dal.Record{}, ItemsToUpdate: []*dal.Record{}}
		if len(in.Items) == 0 {
			return composer.NewStepResponse(out), nil
		}
		byVariant, err := existingVariantItems(ctx, in.ID, in.Items)
		if err != nil {
			return composer.StepResponse[LineItemActions]{}, err
		}
		for _, item := range in.Items {
			existing := FindMatchingLineItem(byVariant[Str(item, "variant_id")], item)
			if existing == nil {
				out.ItemsToCreate = append(out.ItemsToCreate, item)
				continue
			}
			q := totals.Add(0, Get(existing, "quantity"), Coalesce(Get(item, "quantity"), 1))
			existing.Set("quantity", totals.NewBN(q))
			var found *dal.Record
			for _, u := range out.ItemsToUpdate {
				if Get(u, "id") == Get(existing, "id") {
					found = u
				}
			}
			if found != nil {
				found.Set("quantity", totals.NewBN(q))
				continue
			}
			u := dal.NewRecord()
			u.Set("id", Get(existing, "id"))
			u.Set("quantity", totals.NewBN(q))
			u.Set("variant_id", Get(item, "variant_id"))
			u.Set("unit_price", Coalesce(Get(item, "unit_price"), Get(existing, "unit_price")))
			u.Set("compare_at_unit_price", Coalesce(Get(item, "compare_at_unit_price"), Get(existing, "compare_at_unit_price")))
			out.ItemsToUpdate = append(out.ItemsToUpdate, u)
		}
		return composer.NewStepResponse(out), nil
	}, nil)

// CreateLineItemsStep mirrors createLineItemsStep.
var CreateLineItemsStep = composer.CreateStep("create-line-items-step",
	func(ctx composer.StepExecutionContext, in LineItemsInput) (composer.StepResponse[[]*dal.Record], error) {
		created := []*dal.Record{}
		if len(in.Items) > 0 {
			var err error
			created, err = CartOf(ctx.Container).AddLineItems(ctx.Context, in.Items)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := CartOf(ctx.Container).Internal("LineItem").Delete(ctx.Context, ids)
		return err
	})

// UpdateLineItemsStep mirrors updateLineItemsStep: items are { id, ... }
// or { selector: { id }, data }.
var UpdateLineItemsStep = composer.CreateStep("update-line-items-step",
	func(ctx composer.StepExecutionContext, in LineItemsInput) (composer.StepResponse[[]*dal.Record], error) {
		if len(in.Items) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
		}
		svc := CartOf(ctx.Container)
		var ids []any
		var updates []any
		var selects []string
		seen := map[string]bool{"id": true}
		for _, item := range in.Items {
			data := item
			id := Get(item, "id")
			if d := Rec(item, "data"); d != nil {
				data = Copy(d)
				id = Get(Rec(item, "selector"), "id")
			}
			for _, k := range data.Keys() {
				if !seen[k] {
					seen[k] = true
					selects = append(selects, k)
				}
			}
			ids = append(ids, id)
			u := Copy(data)
			u.Set("id", id)
			updates = append(updates, u)
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		before, err := svc.ListBySuffix(ctx.Context, "LineItems", f, utilsmodulessdk.FindConfig{Select: append([]string{"id"}, selects...)})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		res, err := svc.Internal("LineItem").Update(ctx.Context, updates)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated, _ := res.([]*dal.Record)
		if updated == nil {
			updated = []*dal.Record{}
		}
		return composer.NewStepResponse(updated, any(before)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		before, _ := data.([]*dal.Record)
		if len(before) == 0 {
			return nil
		}
		var updates []any
		for _, b := range before {
			updates = append(updates, b)
		}
		_, err := CartOf(ctx.Container).Internal("LineItem").Update(ctx.Context, updates)
		return err
	})
