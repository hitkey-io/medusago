package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// GetItemTaxLinesInput mirrors GetItemTaxLinesStepInput.
type GetItemTaxLinesInput struct {
	OrderOrCart         *dal.Record
	Items               []*dal.Record
	ShippingMethods     []*dal.Record
	ForceTaxCalculation bool
	IsReturn            bool
	ShippingAddress     *dal.Record
	AdditionalContext   *dal.Record
}

// ItemTaxLines mirrors the step output.
type ItemTaxLines struct {
	LineItemTaxLines        []*dal.Record
	ShippingMethodsTaxLines []*dal.Record
}

func g(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	}
	return true
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// normalizeTaxModuleContext mirrors normalizeTaxModuleContext.
func normalizeTaxModuleContext(oc *dal.Record, force, isReturn bool, shippingAddress, additional *dal.Record) (*dal.Record, error) {
	address := shippingAddress
	if address == nil {
		address, _ = g(oc, "shipping_address").(*dal.Record)
	}
	region, _ := g(oc, "region").(*dal.Record)
	if !force && !truthy(g(region, "automatic_taxes")) {
		return nil, nil
	}
	if force && !truthy(g(address, "country_code")) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "country code is required to calculate taxes")
	}
	if !truthy(g(address, "country_code")) {
		return nil, nil
	}
	c := dal.NewRecord()
	a := dal.NewRecord()
	a.Set("country_code", g(address, "country_code"))
	a.Set("province_code", g(address, "province"))
	a.Set("address_1", g(address, "address_1"))
	a.Set("address_2", g(address, "address_2"))
	a.Set("city", g(address, "city"))
	a.Set("postal_code", g(address, "postal_code"))
	a.Set("metadata", g(address, "metadata"))
	c.Set("address", a)
	if customer, ok := g(oc, "customer").(*dal.Record); ok && customer != nil {
		cu := dal.NewRecord()
		cu.Set("id", g(customer, "id"))
		cu.Set("email", g(customer, "email"))
		groups := []any{}
		for _, gr := range records(g(customer, "groups")) {
			groups = append(groups, g(gr, "id"))
		}
		cu.Set("customer_groups", groups)
		cu.Set("metadata", g(customer, "metadata"))
		c.Set("customer", cu)
	} else {
		c.Set("customer", g(oc, "customer"))
	}
	c.Set("is_return", isReturn)
	c.Set("locale", g(oc, "locale"))
	if methods := g(oc, "shipping_methods"); methods != nil {
		var ms []any
		for _, m := range records(methods) {
			x := dal.NewRecord()
			for _, k := range []string{"id", "name", "shipping_option_id", "amount"} {
				x.Set(k, g(m, k))
			}
			ms = append(ms, x)
		}
		if ms == nil {
			ms = []any{}
		}
		c.Set("shipping_methods", ms)
	}
	c.Set("additional_context", additional)
	return c, nil
}

type taxLinesService interface {
	GetTaxLines(ctx context.Context, items []*dal.Record, calculationContext *dal.Record) ([]*dal.Record, error)
}

// GetItemTaxLinesStep mirrors getItemTaxLinesStep.
var GetItemTaxLinesStep = composer.CreateStep("get-item-tax-lines",
	func(ctx composer.StepExecutionContext, in GetItemTaxLinesInput) (composer.StepResponse[ItemTaxLines], error) {
		out := ItemTaxLines{LineItemTaxLines: []*dal.Record{}, ShippingMethodsTaxLines: []*dal.Record{}}
		var filtered []*dal.Record
		for _, item := range in.Items {
			if !truthy(g(item, "is_giftcard")) {
				filtered = append(filtered, item)
			}
		}
		taxCtx, err := normalizeTaxModuleContext(in.OrderOrCart, in.ForceTaxCalculation, in.IsReturn, in.ShippingAddress, in.AdditionalContext)
		if err != nil {
			return composer.StepResponse[ItemTaxLines]{}, err
		}
		if taxCtx == nil {
			return composer.NewStepResponse(out), nil
		}
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.TAX).(taxLinesService)
		currency := g(in.OrderOrCart, "currency_code")
		if len(in.Items) > 0 {
			var items []*dal.Record
			for _, item := range filtered {
				x := dal.NewRecord()
				x.Set("id", g(item, "id"))
				x.Set("product_id", g(item, "product_id"))
				x.Set("product_type_id", g(item, "product_type_id"))
				x.Set("quantity", g(item, "quantity"))
				x.Set("unit_price", g(item, "unit_price"))
				x.Set("currency_code", currency)
				items = append(items, x)
			}
			lines, err := svc.GetTaxLines(ctx.Context, items, taxCtx)
			if err != nil {
				return composer.StepResponse[ItemTaxLines]{}, err
			}
			out.LineItemTaxLines = lines
		}
		if len(in.ShippingMethods) > 0 {
			var items []*dal.Record
			for _, m := range in.ShippingMethods {
				x := dal.NewRecord()
				x.Set("id", g(m, "id"))
				x.Set("shipping_option_id", g(m, "shipping_option_id"))
				x.Set("unit_price", g(m, "amount"))
				x.Set("currency_code", currency)
				items = append(items, x)
			}
			lines, err := svc.GetTaxLines(ctx.Context, items, taxCtx)
			if err != nil {
				return composer.StepResponse[ItemTaxLines]{}, err
			}
			out.ShippingMethodsTaxLines = lines
		}
		return composer.NewStepResponse(out), nil
	}, nil)
