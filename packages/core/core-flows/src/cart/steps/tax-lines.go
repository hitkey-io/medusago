package steps

import (
	"context"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CartService is the cart module surface used by the steps.
type CartService interface {
	commonsteps.ModuleService
	SetLineItemTaxLines(ctx context.Context, cartID string, taxLines []*dal.Record) ([]*dal.Record, error)
	SetShippingMethodTaxLines(ctx context.Context, cartID string, taxLines []*dal.Record) ([]*dal.Record, error)
	AddLineItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
}

// CartOf resolves the cart module service.
func CartOf(c composer.Container) CartService {
	s, _ := c.Resolve(utilsmodulessdk.CART).(CartService)
	return s
}

// SetTaxLinesInput mirrors SetTaxLinesForItemsStepInput.
type SetTaxLinesInput struct {
	Cart             *dal.Record
	ItemTaxLines     []*dal.Record
	ShippingTaxLines []*dal.Record
}

type taxLinesRevert struct {
	cartID                  string
	existingItemLines       []*dal.Record
	existingShippingMethods []*dal.Record
}

func normalizeTaxLines(lines []*dal.Record, ownerKey, ownerFrom string) []*dal.Record {
	out := make([]*dal.Record, len(lines))
	for i, l := range lines {
		r := dal.NewRecord()
		r.Set("description", Get(l, "name"))
		r.Set("tax_rate_id", Get(l, "rate_id"))
		r.Set("code", Get(l, "code"))
		r.Set("rate", Get(l, "rate"))
		r.Set("provider_id", Get(l, "provider_id"))
		r.Set(ownerKey, Get(l, ownerFrom))
		if v, ok := l.Get("data"); ok {
			r.Set("data", v)
		}
		out[i] = r
	}
	return out
}

func listTaxLines(ctx context.Context, svc CartService, suffix, key string, ids []any) ([]*dal.Record, error) {
	if len(ids) == 0 {
		return []*dal.Record{}, nil
	}
	f := dal.NewRecord()
	f.Set(key, ids)
	return svc.ListBySuffix(ctx, suffix, f, utilsmodulessdk.FindConfig{})
}

// SetTaxLinesForItemsStep mirrors setTaxLinesForItemsStep.
var SetTaxLinesForItemsStep = composer.CreateStep("set-tax-lines-for-items",
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
		cartID := Str(in.Cart, "id")
		if _, err := svc.SetLineItemTaxLines(ctx.Context, cartID, normalizeTaxLines(in.ItemTaxLines, "item_id", "line_item_id")); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if _, err := svc.SetShippingMethodTaxLines(ctx.Context, cartID, normalizeTaxLines(in.ShippingTaxLines, "shipping_method_id", "shipping_line_id")); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(taxLinesRevert{cartID, existingItems, existingShip})), nil
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
			if _, err := svc.SetLineItemTaxLines(ctx.Context, r.cartID, restore(r.existingItemLines, "item_id")); err != nil {
				return err
			}
		}
		_, err := svc.SetShippingMethodTaxLines(ctx.Context, r.cartID, restore(r.existingShippingMethods, "shipping_method_id"))
		return err
	})
