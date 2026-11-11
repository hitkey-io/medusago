package steps

import (
	"context"
	"sort"
	"strings"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type pricingService interface {
	CalculatePrices(ctx context.Context, priceSetIDs []any, pricingContext *dal.Record) ([]*dal.Record, error)
}

// PriceItem is one entry of the bulk getVariantPriceSetsStep input.
type PriceItem struct {
	ID        string
	VariantID string
	Context   *dal.Record
}

type calcItem struct {
	id, variantID, priceSetID string
	context                   *dal.Record
}

func contextKey(c *dal.Record) string {
	if c == nil || c.Len() == 0 {
		return "no-context"
	}
	keys := c.Keys()
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		v, _ := c.Get(k)
		parts[i] = k + ":" + JSONString(v)
	}
	return strings.Join(parts, "|")
}

// GetVariantPriceSetsStep mirrors getVariantPriceSetsStep (bulk input):
// the calculated price sets keyed by item id (or variant id).
var GetVariantPriceSetsStep = composer.CreateStep("get-variant-price-sets",
	func(ctx composer.StepExecutionContext, data []PriceItem) (composer.StepResponse[map[string]*dal.Record], error) {
		result := map[string]*dal.Record{}
		if len(data) == 0 {
			return composer.NewStepResponse(result), nil
		}
		ids := make([]any, len(data))
		for i, d := range data {
			ids[i] = d.VariantID
		}
		rows, err := graph(ctx, "variant", []string{"id", "price_set.id"}, idFilter(ids), &caching.CacheOptions{Tags: []string{"LinkProductVariantPriceSet:list:*"}})
		if err != nil {
			return composer.StepResponse[map[string]*dal.Record]{}, err
		}
		custom := map[string]bool{}
		for _, d := range data {
			if Truthy(Get(d.Context, "is_custom_price")) {
				custom[d.VariantID] = true
			}
		}
		var notFound []string
		priceSetOf := map[string]string{}
		for _, r := range Records(rows) {
			id := Str(r, "id")
			ps := Str(Rec(r, "price_set"), "id")
			if ps == "" && !custom[id] {
				notFound = append(notFound, id)
			}
			if ps != "" {
				priceSetOf[id] = ps
			}
		}
		if len(notFound) > 0 {
			return composer.StepResponse[map[string]*dal.Record]{}, invalid("Variants with IDs %s do not have a price", strings.Join(notFound, ", "))
		}
		var items []calcItem
		for _, d := range data {
			if ps, ok := priceSetOf[d.VariantID]; ok {
				items = append(items, calcItem{d.ID, d.VariantID, ps, d.Context})
			}
		}
		var groupKeys []string
		groups := map[string][]calcItem{}
		for _, it := range items {
			k := contextKey(it.context)
			if _, ok := groups[k]; !ok {
				groupKeys = append(groupKeys, k)
			}
			groups[k] = append(groups[k], it)
		}
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.PRICING).(pricingService)
		for _, k := range groupKeys {
			g := groups[k]
			psIDs := make([]any, len(g))
			for i, it := range g {
				psIDs[i] = it.priceSetID
			}
			c := g[0].context
			if c == nil {
				c = dal.NewRecord()
			}
			calculated, err := svc.CalculatePrices(ctx.Context, psIDs, c)
			if err != nil {
				return composer.StepResponse[map[string]*dal.Record]{}, err
			}
			byID := map[string]*dal.Record{}
			for _, ps := range calculated {
				byID[Str(ps, "id")] = ps
			}
			for _, it := range g {
				if ps := byID[it.priceSetID]; ps != nil {
					key := it.id
					if key == "" {
						key = it.variantID
					}
					result[key] = ps
				}
			}
		}
		return composer.NewStepResponse(result), nil
	}, nil)

// ConfirmInventoryStep mirrors confirmInventoryStep.
var ConfirmInventoryStep = composer.CreateStep("confirm-inventory-step",
	func(ctx composer.StepExecutionContext, items []ConfirmInventoryItem) (composer.StepResponse[any], error) {
		if len(items) == 0 {
			return composer.NewStepResponse[any]([]any{}, any([]any{})), nil
		}
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.INVENTORY).(interface {
			ConfirmInventory(ctx context.Context, inventoryItemID any, locationIDs []any, quantity any) (bool, error)
		})
		ok := true
		for _, it := range items {
			if it.AllowBackorder {
				continue
			}
			q := multiply(it.Quantity, it.RequiredQuantity)
			covered, err := svc.ConfirmInventory(ctx.Context, it.InventoryItemID, it.LocationIDs, q)
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			if !covered {
				ok = false
			}
		}
		if !ok {
			return composer.StepResponse[any]{}, notAllowedInventory()
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateCartsStep mirrors createCartsStep.
var CreateCartsStep = composer.CreateStep("create-carts",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.CART).(interface {
			CreateCarts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
		})
		carts, err := svc.CreateCarts(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(carts, any(commonsteps.RecordIDs(carts))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.CART).Internal("Cart").SoftDelete(ctx.Context, ids)
		return err
	})
