package workflows

import (
	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	commonutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/utils"
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var pricingCartFields = append(append([]string{}, CartFieldsForCalculateShippingOptionsPrices...), "sales_channel_id", "item_total", "total")

var commonOptionsFields = []string{"id", "name", "price_type", "service_zone_id", "service_zone.fulfillment_set_id", "service_zone.fulfillment_set.type", "service_zone.fulfillment_set.location.address.*", "shipping_profile_id", "provider_id", "data", "type.id", "type.label", "type.description", "type.code", "provider.id", "provider.is_enabled", "rules.attribute", "rules.value", "rules.operator"}

var flatRateFields = append(append([]string{}, commonOptionsFields...), "calculated_price.*", "prices.*", "prices.price_rules.*")

// PricedOptionsInput mirrors ListShippingOptionsForCartWithPricingWorkflowInput.
type PricedOptionsInput struct {
	CartID         string
	Options        []*dal.Record // { id, data }
	IsReturn       bool
	EnabledInStore *bool
	AdditionalData *dal.Record
}

func getOK(r *dal.Record, k string) (any, bool) {
	if r == nil {
		return nil, false
	}
	return r.Get(k)
}

func shippingAddressFilter(cart *dal.Record) *dal.Record {
	addr := dal.NewRecord()
	sa := Rec(cart, "shipping_address")
	addr.Set("country_code", Get(sa, "country_code"))
	addr.Set("province_code", Get(sa, "province"))
	addr.Set("city", Get(sa, "city"))
	addr.Set("postal_expression", Get(sa, "postal_code"))
	return addr
}

// ListShippingOptionsForCartWithPricingWorkflow mirrors listShippingOptionsForCartWithPricingWorkflow
// (the setShippingOptionsContext and setCalculatedShippingPricingContext hooks have no handlers).
var ListShippingOptionsForCartWithPricingWorkflow = composer.CreateWorkflow("list-shipping-options-for-cart-with-pricing",
	func(e *composer.Execution, in PricedOptionsInput) ([]*dal.Record, error) {
		var optionIDs []any
		for _, o := range in.Options {
			optionIDs = append(optionIDs, Get(o, "id"))
		}
		f := dal.NewRecord()
		f.Set("id", in.CartID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-cart", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "cart", Fields: pricingCartFields, Filters: f},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var cart *dal.Record
		if list := Records(res.Data); len(list) > 0 {
			cart = list[0]
		}
		if _, err := ValidatePresenceOfStep.Run(e, struct {
			entity *dal.Record
			fields []string
		}{cart, []string{"sales_channel_id", "region_id", "currency_code"}}); err != nil {
			return nil, err
		}
		scf := dal.NewRecord()
		scf.Set("id", Get(cart, "sales_channel_id"))
		scRes, err := commonsteps.UseQueryGraphStep.Config("sales_channels-fulfillment-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "sales_channels", Fields: []string{"stock_locations.id", "stock_locations.name", "stock_locations.address.*", "stock_locations.fulfillment_sets.id"}, Filters: scf},
			Options:    joiner.Options{Cache: caching.WithTags(true, commonutils.SalesChannelStockLocationCacheTags...)},
		})
		if err != nil {
			return nil, err
		}
		var setIDs []any
		locationOf := map[any]*dal.Record{}
		if list := Records(scRes.Data); len(list) > 0 {
			for _, sl := range Records(Get(list[0], "stock_locations")) {
				for _, fs := range Records(Get(sl, "fulfillment_sets")) {
					id := Get(fs, "id")
					if _, ok := locationOf[id]; !ok {
						setIDs = append(setIDs, id)
					}
					locationOf[id] = sl
				}
			}
		}
		if setIDs == nil {
			setIDs = []any{}
		}
		ctxRec := dal.NewRecord()
		if in.IsReturn {
			ctxRec.Set("is_return", "true")
		} else {
			ctxRec.Set("is_return", "false")
		}
		if in.EnabledInStore == nil || *in.EnabledInStore {
			ctxRec.Set("enabled_in_store", "true")
		} else {
			ctxRec.Set("enabled_in_store", "false")
		}
		filters := dal.NewRecord()
		if len(optionIDs) > 0 {
			filters.Set("id", optionIDs)
		}
		filters.Set("fulfillment_set_id", setIDs)
		filters.Set("address", shippingAddressFilter(cart))
		initial, err := commonsteps.UseQueryGraphStep.Config("shipping-options-price-type-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "shipping_options", Fields: []string{"id", "price_type"}, Filters: filters, Context: query.QueryContext(ctxRec)},
			Options:    joiner.Options{Cache: caching.Enabled()},
		})
		if err != nil {
			return nil, err
		}
		var flatIDs, calcIDs []any
		for _, o := range Records(initial.Data) {
			if Get(o, "price_type") == "flat" {
				flatIDs = append(flatIDs, Get(o, "id"))
			} else {
				calcIDs = append(calcIDs, Get(o, "id"))
			}
		}
		if flatIDs == nil {
			flatIDs = []any{}
		}
		vars := dal.NewRecord()
		vars.Set("context", ctxRec)
		vars.Set("filters", filters)
		vars.Set("id", flatIDs)
		cp := dal.NewRecord()
		cp.Set("context", cart)
		vars.Set("calculated_price", cp)
		flat, err := commonsteps.UseRemoteQueryStep.Config("shipping-options-query-flat-rate", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{EntryPoint: "shipping_options", Fields: flatRateFields, Variables: vars})
		if err != nil {
			return nil, err
		}
		var calculated []*dal.Record
		var prices []*dal.Record
		if len(calcIDs) > 0 {
			cvars := dal.NewRecord()
			cvars.Set("context", ctxRec)
			cvars.Set("filters", filters)
			cvars.Set("id", calcIDs)
			calc, err := commonsteps.UseRemoteQueryStep.Config("shipping-options-query-calculated", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{EntryPoint: "shipping_options", Fields: commonOptionsFields, Variables: cvars})
			if err != nil {
				return nil, err
			}
			calculated = Records(calc.Data)
			dataOf := map[any]any{}
			for _, o := range in.Options {
				dataOf[Get(o, "id")] = Get(o, "data")
			}
			var input []*dal.Record
			for _, so := range calculated {
				c := dal.NewRecord()
				for _, k := range cart.Keys() {
					c.Set(k, Get(cart, k))
				}
				c.Set("items", FilterCartItemsByShippingProfile(Records(Get(cart, "items")), Get(so, "shipping_profile_id")))
				if loc, ok := locationOf[Get(Rec(so, "service_zone"), "fulfillment_set_id")]; ok {
					c.Set("from_location", loc)
				}
				d := dal.NewRecord()
				d.Set("id", Get(so, "id"))
				d.Set("optionData", Get(so, "data"))
				d.Set("context", c)
				if v := dataOf[Get(so, "id")]; v != nil {
					d.Set("data", v)
				}
				d.Set("provider_id", Get(so, "provider_id"))
				input = append(input, d)
			}
			if input == nil {
				input = []*dal.Record{}
			}
			prices, err = fulfillmentsteps.CalculateShippingOptionsPricesStep.Run(e, input)
			if err != nil {
				return nil, err
			}
		}
		var out []*dal.Record
		for _, o := range Records(flat.Data) {
			price := Rec(o, "calculated_price")
			r := Copy(o)
			var amount any
			defined := false
			if price != nil {
				amount, defined = price.Get("calculated_amount")
			}
			if defined {
				r.Set("amount", amount)
			} else {
				r.Delete("amount")
			}
			r.Set("is_tax_inclusive", Truthy(Get(price, "is_calculated_price_tax_inclusive")))
			r.Set("stock_location", locationOf[Get(Rec(o, "service_zone"), "fulfillment_set_id")])
			out = append(out, r)
		}
		for i, o := range calculated {
			var price *dal.Record
			if i < len(prices) {
				price = prices[i]
			}
			r := Copy(o)
			if v, ok := getOK(price, "calculated_amount"); ok {
				r.Set("amount", v)
			} else {
				r.Delete("amount")
			}
			if v, ok := getOK(price, "is_calculated_price_tax_inclusive"); ok {
				r.Set("is_tax_inclusive", v)
			} else {
				r.Delete("is_tax_inclusive")
			}
			if price != nil {
				r.Set("calculated_price", price)
			} else {
				r.Delete("calculated_price")
			}
			r.Set("stock_location", locationOf[Get(Rec(o, "service_zone"), "fulfillment_set_id")])
			out = append(out, r)
		}
		if out == nil {
			out = []*dal.Record{}
		}
		return commonsteps.GetTranslatedShippingOptionsStep.Run(e, commonsteps.GetTranslatedShippingOptionsStepInput{ShippingOptions: out, Locale: Str(cart, "locale")})
	})
