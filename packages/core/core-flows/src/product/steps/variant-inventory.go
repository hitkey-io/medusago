package steps

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// VariantIDsInput mirrors { variantIds }.
type VariantIDsInput struct {
	VariantIDs []any
}

func dedupe(ids []any) []any {
	seen := map[string]bool{}
	var out []any
	for _, id := range ids {
		if id == nil || id == "" {
			continue
		}
		k := fmt.Sprint(id)
		if !seen[k] {
			seen[k] = true
			out = append(out, id)
		}
	}
	return out
}

func variantInventoryLinks(ctx composer.StepExecutionContext, variantIDs []any) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("variant_id", variantIDs)
	res, err := commonsteps.QueryOf(ctx.Container).Graph(ctx.Context, query.GraphInput{Entity: "product_variant_inventory_item", Fields: []string{"inventory_item_id", "variant_id"}, Filters: f}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	var out []*dal.Record
	for _, x := range res.Data {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out, nil
}

func variantInventoryLink(variantID, itemID any, required bool) *dal.Record {
	a := dal.NewRecord()
	a.Set("variant_id", variantID)
	b := dal.NewRecord()
	b.Set("inventory_item_id", itemID)
	out := dal.NewRecord()
	out.Set(utilsmodulessdk.PRODUCT, a)
	out.Set(utilsmodulessdk.INVENTORY, b)
	if required {
		d := dal.NewRecord()
		d.Set("required_quantity", float64(1))
		out.Set("data", d)
	}
	return out
}

type dismissedInventory struct {
	order []string
	items map[string][]any
}

// DismissProductVariantsInventoryStep mirrors
// dismissProductVariantsInventoryStep.
var DismissProductVariantsInventoryStep = composer.CreateStep("dismiss-product-variants-inventory",
	func(ctx composer.StepExecutionContext, in VariantIDsInput) (composer.StepResponse[any], error) {
		if len(in.VariantIDs) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		rows, err := variantInventoryLinks(ctx, in.VariantIDs)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		byVariant := map[string][]any{}
		for _, r := range rows {
			vid, _ := r.Get("variant_id")
			iid, _ := r.Get("inventory_item_id")
			byVariant[fmt.Sprint(vid)] = append(byVariant[fmt.Sprint(vid)], iid)
		}
		dismissed := dismissedInventory{items: map[string][]any{}}
		var links []*dal.Record
		for _, vid := range in.VariantIDs {
			if vid == nil || vid == "" {
				continue
			}
			k := fmt.Sprint(vid)
			dismissed.order = append(dismissed.order, k)
			dismissed.items[k] = byVariant[k]
			for _, iid := range byVariant[k] {
				links = append(links, variantInventoryLink(vid, iid, false))
			}
		}
		if _, err := commonsteps.LinkOf(ctx.Container).Dismiss(ctx.Context, links); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(dismissed)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		d, ok := data.(dismissedInventory)
		if !ok {
			return nil
		}
		var links []*dal.Record
		for _, vid := range d.order {
			for _, iid := range d.items[vid] {
				links = append(links, variantInventoryLink(vid, iid, false))
			}
		}
		if len(links) == 0 {
			return nil
		}
		_, err := commonsteps.LinkOf(ctx.Container).Create(ctx.Context, links)
		return err
	})

type createdDefaultInventory struct {
	itemIDs []any
	links   []*dal.Record
}

// CreateProductVariantsDefaultInventoryStep mirrors
// createProductVariantsDefaultInventoryStep.
var CreateProductVariantsDefaultInventoryStep = composer.CreateStep("create-product-variants-default-inventory",
	func(ctx composer.StepExecutionContext, in VariantIDsInput) (composer.StepResponse[any], error) {
		ids := dedupe(in.VariantIDs)
		if len(ids) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		rows, err := variantInventoryLinks(ctx, ids)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		has := map[string]bool{}
		for _, r := range rows {
			vid, _ := r.Get("variant_id")
			has[fmt.Sprint(vid)] = true
		}
		var without []any
		for _, id := range ids {
			if !has[fmt.Sprint(id)] {
				without = append(without, id)
			}
		}
		if len(without) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		f := dal.NewRecord()
		f.Set("id", without)
		variants, err := ProductService(ctx.Container).ListBySuffix(ctx.Context, "ProductVariants", f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if len(variants) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		inv := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.INVENTORY)
		toCreate := make([]*dal.Record, len(variants))
		for i, v := range variants {
			toCreate[i] = utils.CreateDefaultInventoryItem(v)
		}
		items, err := inv.Internal("InventoryItem").Create(ctx.Context, toCreate)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		links := make([]*dal.Record, len(variants))
		itemIDs := make([]any, len(items))
		for i, v := range variants {
			vid, _ := v.Get("id")
			iid, _ := items[i].Get("id")
			itemIDs[i] = iid
			links[i] = variantInventoryLink(vid, iid, true)
		}
		if _, err := commonsteps.LinkOf(ctx.Container).Create(ctx.Context, links); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(createdDefaultInventory{itemIDs: itemIDs, links: links})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(createdDefaultInventory)
		if !ok || len(c.itemIDs) == 0 {
			return nil
		}
		if _, err := commonsteps.LinkOf(ctx.Container).Dismiss(ctx.Context, c.links); err != nil {
			return err
		}
		inv := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.INVENTORY)
		_, err := inv.Internal("InventoryItem").Delete(ctx.Context, c.itemIDs)
		return err
	})

// VariantPriceSetLink is { variant_id, price_set_id }.
type PriceSetLinkRow struct {
	VariantID  any
	PriceSetID any
}

// GetVariantPricingLinkStep mirrors getVariantPricingLinkStep.
var GetVariantPricingLinkStep = composer.CreateStep("get-variant-pricing-link",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]PriceSetLinkRow], error) {
		if len(ids) == 0 {
			return composer.NewStepResponse([]PriceSetLinkRow{}), nil
		}
		svc := commonsteps.LinkOf(ctx.Container).GetLinkModule(utilsmodulessdk.PRODUCT, "variant_id", utilsmodulessdk.PRICING, "price_set_id")
		f := dal.NewRecord()
		f.Set("variant_id", ids)
		rows, err := svc.List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: []string{"variant_id", "price_set_id"}})
		if err != nil {
			return composer.StepResponse[[]PriceSetLinkRow]{}, err
		}
		out := make([]PriceSetLinkRow, len(rows))
		found := map[string]bool{}
		for i, r := range rows {
			v, _ := r.Get("variant_id")
			p, _ := r.Get("price_set_id")
			out[i] = PriceSetLinkRow{VariantID: v, PriceSetID: p}
			found[fmt.Sprint(v)] = true
		}
		if len(rows) != len(ids) {
			var missing []string
			for _, id := range ids {
				if !found[fmt.Sprint(id)] {
					missing = append(missing, fmt.Sprint(id))
				}
			}
			return composer.StepResponse[[]PriceSetLinkRow]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Variants with IDs %s do not have prices associated.", joinStrings(missing)))
		}
		return composer.NewStepResponse(out), nil
	}, nil)

func joinStrings(s []string) string {
	out := ""
	for i, x := range s {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}
