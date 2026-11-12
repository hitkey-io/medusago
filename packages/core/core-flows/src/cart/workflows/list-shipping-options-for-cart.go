package workflows

import (
	commonutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/utils"
	"strings"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var listOptionsCartFields = append(append([]string{}, FieldsForPricingContext...), "items.*", "items.variant.manage_inventory", "items.variant.allow_backorder", "items.variant.inventory_items.inventory_item_id", "items.variant.inventory_items.inventory.requires_shipping", "items.variant.inventory_items.inventory.location_levels.*")

// ListShippingOptionsInput mirrors ListShippingOptionsForCartWorkflowInput.
type ListShippingOptionsInput struct {
	CartID         string
	OptionIDs      []any
	IsReturn       bool
	EnabledInStore *bool
	Fields         []string
	AdditionalData *dal.Record
}

// ValidatePresenceOfStep mirrors validatePresenceOfStep.
var ValidatePresenceOfStep = composer.CreateStep("validate-presence-of",
	func(ctx composer.StepExecutionContext, in struct {
		entity *dal.Record
		fields []string
	}) (composer.StepResponse[any], error) {
		var missing []string
		for _, f := range in.fields {
			if !IsPresent(Get(in.entity, f)) {
				missing = append(missing, f)
			}
		}
		if len(missing) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Field(s) are required to have value to continue - "+strings.Join(missing, ", "))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// mergeRecord mirrors spreading a hook result into an object (a result
// that is not an object adds nothing).
func mergeRecord(dst *dal.Record, v any) {
	r, ok := v.(*dal.Record)
	if !ok || r == nil {
		return
	}
	for _, k := range r.Keys() {
		x, _ := r.Get(k)
		dst.Set(k, x)
	}
}

// ListShippingOptionsForCartWorkflow mirrors listShippingOptionsForCartWorkflow.
var ListShippingOptionsForCartWorkflow = composer.CreateWorkflow("list-shipping-options-for-cart",
	func(e *composer.Execution, in ListShippingOptionsInput) ([]*dal.Record, error) {
		f := dal.NewRecord()
		f.Set("id", in.CartID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-cart", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "cart", Fields: listOptionsCartFields, Filters: f},
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
			GraphInput: query.GraphInput{Entity: "sales_channels", Fields: []string{"stock_locations.fulfillment_sets.id", "stock_locations.id", "stock_locations.name", "stock_locations.address.*"}, Filters: scf},
			Options:    joiner.Options{Cache: caching.WithTags(true, commonutils.SalesChannelStockLocationCacheTags...)},
		})
		if err != nil {
			return nil, err
		}
		var setIDs []any
		seen := map[any]bool{}
		if list := Records(scRes.Data); len(list) > 0 {
			for _, sl := range Records(Get(list[0], "stock_locations")) {
				for _, fs := range Records(Get(sl, "fulfillment_sets")) {
					if id := Get(fs, "id"); !seen[id] {
						seen[id] = true
						setIDs = append(setIDs, id)
					}
				}
			}
		}
		if setIDs == nil {
			setIDs = []any{}
		}
		hookIn := dal.NewRecord()
		hookIn.Set("cart", cart)
		hookIn.Set("fulfillmentSetIds", setIDs)
		hookIn.Set("additional_data", in.AdditionalData)
		pricingResult, err := composer.CreateHookWithResult(e, "setPricingContext", hookIn)
		if err != nil {
			return nil, err
		}
		optionsResult, err := composer.CreateHookWithResult(e, "setShippingOptionsContext", hookIn)
		if err != nil {
			return nil, err
		}
		vars := dal.NewRecord()
		if in.OptionIDs != nil {
			vars.Set("id", in.OptionIDs)
		}
		// { ...setShippingOptionsContextResult, is_return, enabled_in_store }
		c := dal.NewRecord()
		mergeRecord(c, optionsResult)
		if in.IsReturn {
			c.Set("is_return", "true")
		} else {
			c.Set("is_return", "false")
		}
		if in.EnabledInStore == nil || *in.EnabledInStore {
			c.Set("enabled_in_store", "true")
		} else {
			c.Set("enabled_in_store", "false")
		}
		vars.Set("context", c)
		filters := dal.NewRecord()
		filters.Set("fulfillment_set_id", setIDs)
		addr := dal.NewRecord()
		sa := Rec(cart, "shipping_address")
		addr.Set("country_code", Get(sa, "country_code"))
		addr.Set("province_code", Get(sa, "province"))
		addr.Set("city", Get(sa, "city"))
		addr.Set("postal_expression", Get(sa, "postal_code"))
		filters.Set("address", addr)
		vars.Set("filters", filters)
		// { ...cart pricing fields, ...setPricingContextResult, currency_code, ... }
		pc := FilterObjectByKeys(cart, FieldsForPricingContext)
		mergeRecord(pc, pricingResult)
		for _, k := range []string{"currency_code", "region_id", "region", "customer_id", "customer"} {
			if v, ok := cart.Get(k); ok {
				pc.Set(k, v)
			} else {
				pc.Delete(k)
			}
		}
		cp := dal.NewRecord()
		cp.Set("context", pc)
		vars.Set("calculated_price", cp)
		fields := Deduplicate(in.Fields, []string{"id", "name", "price_type", "service_zone_id", "shipping_profile_id", "provider_id", "data", "service_zone.fulfillment_set_id", "service_zone.fulfillment_set.type", "service_zone.fulfillment_set.location.id", "service_zone.fulfillment_set.location.address.*", "type.id", "type.label", "type.description", "type.code", "provider.id", "provider.is_enabled", "rules.attribute", "rules.value", "rules.operator", "calculated_price.*", "prices.*", "prices.price_rules.*"})
		opts, err := commonsteps.UseRemoteQueryStep.Config("shipping-options-query", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{EntryPoint: "shipping_options", Fields: fields, Variables: vars})
		if err != nil {
			return nil, err
		}
		items := Records(Get(cart, "items"))
		var out []*dal.Record
		for _, o := range Records(opts.Data) {
			price := Rec(o, "calculated_price")
			locationID := Get(Rec(Rec(Rec(o, "service_zone"), "fulfillment_set"), "location"), "id")
			insufficient := false
			for _, item := range items {
				variant := Rec(item, "variant")
				if !Truthy(Get(variant, "manage_inventory")) || Truthy(Get(variant, "allow_backorder")) {
					continue
				}
				for _, ii := range Records(Get(variant, "inventory_items")) {
					inv := Rec(ii, "inventory")
					if !Truthy(Get(inv, "requires_shipping")) {
						continue
					}
					var level *dal.Record
					for _, l := range Records(Get(inv, "location_levels")) {
						if Get(l, "location_id") == locationID {
							level = l
							break
						}
					}
					if level == nil || totals.Lt(Get(level, "available_quantity"), Get(item, "quantity")) {
						insufficient = true
					}
				}
			}
			r := Copy(o)
			var amount any
			defined := false
			if price != nil {
				amount, defined = price.Get("calculated_amount")
			}
			if defined {
				r.Set("amount", amount)
			} else {
				// price?.calculated_amount is undefined: JSON omits the key
				r.Delete("amount")
			}
			r.Set("is_tax_inclusive", Truthy(Get(price, "is_calculated_price_tax_inclusive")))
			r.Set("insufficient_inventory", insufficient)
			out = append(out, r)
		}
		if out == nil {
			out = []*dal.Record{}
		}
		return commonsteps.GetTranslatedShippingOptionsStep.Run(e, commonsteps.GetTranslatedShippingOptionsStepInput{ShippingOptions: out, Locale: Str(cart, "locale")})
	})
