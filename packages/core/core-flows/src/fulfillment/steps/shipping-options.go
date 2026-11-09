package steps

import (
	"context"
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	pricingservices "github.com/hitkey-io/medusago/packages/modules/pricing/src/services"
)

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
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

func pricing(c composer.Container) *pricingservices.PricingModuleService {
	s, _ := c.Resolve(utilsmodulessdk.PRICING).(*pricingservices.PricingModuleService)
	return s
}

// ValidateFulfillmentProvidersStep mirrors validateFulfillmentProvidersStep.
var ValidateFulfillmentProvidersStep = composer.CreateStep("validate-fulfillment-providers-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[any], error) {
		svc := Fulfillment(ctx.Container)
		var optionIDs []any
		for _, d := range input {
			if id := get(d, "id"); id != nil && id != "" {
				optionIDs = append(optionIDs, id)
			}
		}
		if optionIDs == nil {
			optionIDs = []any{}
		}
		f := dal.NewRecord()
		f.Set("id", optionIDs)
		options, err := svc.Internal("ShippingOption").List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: []string{"id", "service_zone_id", "provider_id"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		optMap := map[string]*dal.Record{}
		for _, o := range options {
			optMap[fmt.Sprint(get(o, "id"))] = o
		}
		type pair struct{ zone, provider any }
		var toValidate []pair
		for _, d := range input {
			if _, has := d.Get("id"); has {
				existing := optMap[fmt.Sprint(get(d, "id"))]
				if !truthy(get(d, "service_zone_id")) {
					d.Set("service_zone_id", get(existing, "service_zone_id"))
				}
				if !truthy(get(d, "provider_id")) {
					d.Set("provider_id", get(existing, "provider_id"))
				}
				toValidate = append(toValidate, pair{get(d, "service_zone_id"), get(d, "provider_id")})
				continue
			}
			if truthy(get(d, "service_zone_id")) && truthy(get(d, "provider_id")) {
				toValidate = append(toValidate, pair{get(d, "service_zone_id"), get(d, "provider_id")})
				continue
			}
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "service_zone_id and provider_id are required to create a shipping option")
		}
		zoneIDs := make([]any, len(input))
		for i, d := range input {
			zoneIDs[i] = get(d, "service_zone_id")
		}
		zones, err := remoteQueryByID(ctx, "service_zone", zoneIDs, []string{"id", "fulfillment_set.locations.fulfillment_providers.id"})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		zoneMap := map[string]*dal.Record{}
		for _, z := range zones {
			zoneMap[fmt.Sprint(get(z, "id"))] = z
		}
		var invalid []string
		for _, p := range toValidate {
			zone := zoneMap[fmt.Sprint(p.zone)]
			var providers []string
			if fs, ok := get(zone, "fulfillment_set").(*dal.Record); ok {
				for _, loc := range records(get(fs, "locations")) {
					for _, fp := range records(get(loc, "fulfillment_providers")) {
						providers = append(providers, fmt.Sprint(get(fp, "id")))
					}
				}
			}
			found := false
			for _, pr := range providers {
				if pr == fmt.Sprint(p.provider) {
					found = true
				}
			}
			if !found {
				// Array.join renders undefined/null as ""
				if p.provider == nil {
					invalid = append(invalid, "")
				} else {
					invalid = append(invalid, fmt.Sprint(p.provider))
				}
			}
		}
		if len(invalid) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Providers (%s) are not enabled for the service location", strings.Join(invalid, ",")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	}
	return true
}

type remoteQuerier interface {
	RemoteQuery(ctx context.Context, entryPoint string, variables *dal.Record, fields []string) ([]any, error)
}

// remoteQueryByID mirrors remoteQuery({ entryPoint, fields, variables: { id } }).
func remoteQueryByID(ctx composer.StepExecutionContext, entryPoint string, ids []any, fields []string) ([]*dal.Record, error) {
	vars := dal.NewRecord()
	vars.Set("id", ids)
	data, err := commonsteps.RemoteQuery(ctx, entryPoint, vars, fields)
	if err != nil {
		return nil, err
	}
	return records(data), nil
}

// ValidateShippingOptionPricesStep mirrors validateShippingOptionPricesStep.
var ValidateShippingOptionPricesStep = composer.CreateStep("validate-shipping-option-prices",
	func(ctx composer.StepExecutionContext, options []*dal.Record) (composer.StepResponse[any], error) {
		svc := Fulfillment(ctx.Container)
		var optionIDs []any
		for _, o := range options {
			optionIDs = append(optionIDs, get(o, "id"))
		}
		if len(optionIDs) > 0 {
			var nonNil []any
			for _, id := range optionIDs {
				if id != nil {
					nonNil = append(nonNil, id)
				}
			}
			if nonNil == nil {
				nonNil = []any{}
			}
			f := dal.NewRecord()
			f.Set("id", nonNil)
			existing, err := svc.Internal("ShippingOption").List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: []string{"id", "price_type", "provider_id"}})
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			m := map[string]*dal.Record{}
			for _, e := range existing {
				m[fmt.Sprint(get(e, "id"))] = e
			}
			for _, o := range options {
				ex := m[fmt.Sprint(get(o, "id"))]
				if get(o, "price_type") == nil {
					if v := get(ex, "price_type"); v != nil {
						o.Set("price_type", v)
					}
				}
				if get(o, "provider_id") == nil {
					if v := get(ex, "provider_id"); v != nil {
						o.Set("provider_id", v)
					}
				}
			}
		}
		var flatPrices []*dal.Record
		var calculated []*dal.Record
		for _, o := range options {
			switch get(o, "price_type") {
			case "flat":
				flatPrices = append(flatPrices, records(get(o, "prices"))...)
			case "calculated":
				calculated = append(calculated, o)
			}
		}
		if len(calculated) > 0 {
			// the manual provider cannot calculate prices
			names := make([]string, len(calculated))
			for i, o := range calculated {
				names[i] = fmt.Sprint(get(o, "name"))
			}
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Cannot calcuate pricing for: [%s] shipping option(s).", strings.Join(names, ", ")))
		}
		var regionIDs []any
		seen := map[string]bool{}
		for _, p := range flatPrices {
			if r := get(p, "region_id"); truthy(r) && !seen[fmt.Sprint(r)] {
				seen[fmt.Sprint(r)] = true
				regionIDs = append(regionIDs, r)
			}
		}
		if len(regionIDs) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		regions := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.REGION)
		f := dal.NewRecord()
		f.Set("id", regionIDs)
		list, err := regions.ListBySuffix(ctx.Context, "Regions", f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if len(list) != len(regionIDs) {
			found := map[string]bool{}
			for _, r := range list {
				found[fmt.Sprint(get(r, "id"))] = true
			}
			var missing []string
			for _, id := range regionIDs {
				if !found[fmt.Sprint(id)] {
					missing = append(missing, fmt.Sprint(id))
				}
			}
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Cannot create prices for non-existent regions. Region with ids [%s] were not found.", strings.Join(missing, ", ")))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

type upsertCompensation struct {
	updatedPrev []*dal.Record
	createdIDs  []any
}

// UpsertShippingOptionsStep mirrors upsertShippingOptionsStep.
var UpsertShippingOptionsStep = composer.CreateStep("create-shipping-options-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(input) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, any(upsertCompensation{})), nil
		}
		svc := Fulfillment(ctx.Container)
		var toUpdate []*dal.Record
		for _, in := range input {
			if truthy(get(in, "id")) {
				toUpdate = append(toUpdate, in)
			}
		}
		var prev []*dal.Record
		if len(toUpdate) > 0 {
			f := dal.NewRecord()
			ids := make([]any, len(toUpdate))
			for i, u := range toUpdate {
				ids[i] = get(u, "id")
			}
			f.Set("id", ids)
			var err error
			prev, err = commonsteps.PrevDataFor(ctx.Context, svc, "ShippingOptions", f, toUpdate...)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
		}
		data := make([]*dal.Record, len(input))
		for i, in := range input {
			n := dal.NewRecord()
			for _, k := range in.Keys() {
				v, _ := in.Get(k)
				n.Set(k, v)
			}
			if t := get(in, "type_id"); truthy(t) {
				n.Set("type", t)
			}
			data[i] = n
		}
		out, err := svc.UpsertShippingOptions(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		updated := map[string]bool{}
		for _, u := range toUpdate {
			updated[fmt.Sprint(get(u, "id"))] = true
		}
		var created []any
		for _, o := range out {
			if !updated[fmt.Sprint(get(o, "id"))] {
				created = append(created, get(o, "id"))
			}
		}
		return composer.NewStepResponse(out, any(upsertCompensation{updatedPrev: prev, createdIDs: created})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(upsertCompensation)
		if !ok {
			return nil
		}
		svc := Fulfillment(ctx.Container)
		if len(c.updatedPrev) > 0 {
			if _, err := svc.UpsertShippingOptions(ctx.Context, c.updatedPrev); err != nil {
				return err
			}
		}
		if len(c.createdIDs) > 0 {
			_, err := svc.Internal("ShippingOption").Delete(ctx.Context, c.createdIDs)
			return err
		}
		return nil
	})

func regionCurrencies(ctx composer.StepExecutionContext, prices []*dal.Record) (map[string]any, error) {
	var ids []any
	seen := map[string]bool{}
	for _, p := range prices {
		if r, ok := p.Get("region_id"); ok && !seen[fmt.Sprint(r)] {
			seen[fmt.Sprint(r)] = true
			ids = append(ids, r)
		}
	}
	out := map[string]any{}
	if len(ids) == 0 {
		return out, nil
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	regions, err := commonsteps.ModuleOf(ctx.Container, utilsmodulessdk.REGION).ListBySuffix(ctx.Context, "Regions", f, utilsmodulessdk.FindConfig{Select: []string{"id", "currency_code"}})
	if err != nil {
		return nil, err
	}
	for _, r := range regions {
		out[fmt.Sprint(get(r, "id"))] = get(r, "currency_code")
	}
	return out, nil
}

func additionalRules(price *dal.Record) *dal.Record {
	rules := dal.NewRecord()
	for _, rule := range records(get(price, "rules")) {
		attr := fmt.Sprint(get(rule, "attribute"))
		existing, _ := rules.Get(attr)
		if _, isStr := existing.(string); isStr {
			continue
		}
		list, _ := existing.([]any)
		r := dal.NewRecord()
		r.Set("operator", get(rule, "operator"))
		r.Set("value", get(rule, "value"))
		rules.Set(attr, append(list, r))
	}
	return rules
}

// buildPriceSet mirrors buildPriceSet.
func buildPriceSet(prices []*dal.Record, currencies map[string]any) *dal.Record {
	out := make([]any, len(prices))
	for i, price := range prices {
		extra := additionalRules(price)
		p := dal.NewRecord()
		if cc, ok := price.Get("currency_code"); ok {
			p.Set("currency_code", cc)
			p.Set("amount", get(price, "amount"))
			p.Set("rules", extra)
		} else {
			region := get(price, "region_id")
			p.Set("currency_code", currencies[fmt.Sprint(region)])
			p.Set("amount", get(price, "amount"))
			rules := dal.NewRecord()
			rules.Set("region_id", region)
			for _, k := range extra.Keys() {
				v, _ := extra.Get(k)
				rules.Set(k, v)
			}
			p.Set("rules", rules)
		}
		out[i] = p
	}
	set := dal.NewRecord()
	set.Set("prices", out)
	return set
}

// OptionPrices is { id, prices }.
type OptionPrices struct {
	ID     any
	Prices []*dal.Record
	// Defined is false when prices was undefined (update without prices).
	Defined bool
}

// OptionPriceSet is { id, priceSetId }.
type OptionPriceSet struct {
	ID         any
	PriceSetID any
}

// CreateShippingOptionsPriceSetsStep mirrors createShippingOptionsPriceSetsStep.
var CreateShippingOptionsPriceSetsStep = composer.CreateStep("add-shipping-options-prices-step",
	func(ctx composer.StepExecutionContext, data []OptionPrices) (composer.StepResponse[[]OptionPriceSet], error) {
		if len(data) == 0 {
			return composer.NewStepResponse([]OptionPriceSet{}, any([]any{})), nil
		}
		var all []*dal.Record
		for _, d := range data {
			all = append(all, d.Prices...)
		}
		currencies, err := regionCurrencies(ctx, all)
		if err != nil {
			return composer.StepResponse[[]OptionPriceSet]{}, err
		}
		sets := make([]*dal.Record, len(data))
		for i, d := range data {
			sets[i] = buildPriceSet(d.Prices, currencies)
		}
		created, err := pricing(ctx.Container).CreatePriceSets(ctx.Context, sets)
		if err != nil {
			return composer.StepResponse[[]OptionPriceSet]{}, err
		}
		out := make([]OptionPriceSet, len(data))
		ids := make([]any, len(created))
		for i, d := range data {
			out[i] = OptionPriceSet{ID: d.ID, PriceSetID: get(created[i], "id")}
			ids[i] = get(created[i], "id")
		}
		return composer.NewStepResponse(out, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := pricing(ctx.Container).Internal("PriceSet").Delete(ctx.Context, ids)
		return err
	})

func optionPriceLink(optionID, priceSetID any) *dal.Record {
	return modulessdk.NewLinkDefinition("fulfillment", "shipping_option_id", optionID, "pricing", "price_set_id", priceSetID)
}

type linkSwap struct{ create, remove []*dal.Record }

// SetShippingOptionsPriceSetsStep mirrors setShippingOptionsPriceSetsStep.
var SetShippingOptionsPriceSetsStep = composer.CreateStep("set-shipping-options-price-sets-step",
	func(ctx composer.StepExecutionContext, data []OptionPriceSet) (composer.StepResponse[any], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		optionIDs := make([]any, len(data))
		for i, d := range data {
			optionIDs[i] = d.ID
		}
		link := commonsteps.LinkOf(ctx.Container)
		svc := link.GetLinkModule(utilsmodulessdk.FULFILLMENT, "shipping_option_id", utilsmodulessdk.PRICING, "price_set_id")
		f := dal.NewRecord()
		f.Set("shipping_option_id", optionIDs)
		existing, err := svc.List(ctx.Context, f, utilsmodulessdk.FindConfig{Select: []string{"shipping_option_id", "price_set_id"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		var toRemove, toCreate []*dal.Record
		for _, l := range existing {
			keep := false
			for _, d := range data {
				if fmt.Sprint(d.ID) == fmt.Sprint(get(l, "shipping_option_id")) && fmt.Sprint(d.PriceSetID) == fmt.Sprint(get(l, "price_set_id")) {
					keep = true
				}
			}
			if !keep {
				toRemove = append(toRemove, optionPriceLink(get(l, "shipping_option_id"), get(l, "price_set_id")))
			}
		}
		for _, d := range data {
			exists := false
			for _, l := range existing {
				if fmt.Sprint(d.ID) == fmt.Sprint(get(l, "shipping_option_id")) && fmt.Sprint(d.PriceSetID) == fmt.Sprint(get(l, "price_set_id")) {
					exists = true
				}
			}
			if !exists {
				toCreate = append(toCreate, optionPriceLink(d.ID, d.PriceSetID))
			}
		}
		if len(toRemove) > 0 {
			if _, err := link.Dismiss(ctx.Context, toRemove); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		if len(toCreate) > 0 {
			if _, err := link.Create(ctx.Context, toCreate); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(linkSwap{create: toRemove, remove: toCreate})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		s, ok := data.(linkSwap)
		if !ok {
			return nil
		}
		link := commonsteps.LinkOf(ctx.Container)
		if len(s.remove) > 0 {
			if _, err := link.Dismiss(ctx.Context, s.remove); err != nil {
				return err
			}
		}
		if len(s.create) > 0 {
			_, err := link.Create(ctx.Context, s.create)
			return err
		}
		return nil
	})

type currentPrices struct {
	optionID, priceSetID any
	prices               []any
}

// SetShippingOptionsPricesStep mirrors setShippingOptionsPricesStep.
var SetShippingOptionsPricesStep = composer.CreateStep("set-shipping-options-prices-step",
	func(ctx composer.StepExecutionContext, data []OptionPrices) (composer.StepResponse[any], error) {
		if len(data) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		var all []*dal.Record
		for _, d := range data {
			all = append(all, d.Prices...)
		}
		currencies, err := regionCurrencies(ctx, all)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		optionIDs := make([]any, len(data))
		for i, d := range data {
			optionIDs[i] = d.ID
		}
		vars := dal.NewRecord()
		f := dal.NewRecord()
		f.Set("shipping_option_id", optionIDs)
		vars.Set("filters", f)
		rows, err := commonsteps.RemoteQueryService(ctx, "shipping_option_price_set", vars, []string{"shipping_option_id", "price_set_id", "price_set.prices.*"})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		var current []currentPrices
		for _, r := range records(rows) {
			var prices []any
			if ps, ok := get(r, "price_set").(*dal.Record); ok {
				prices, _ = get(ps, "prices").([]any)
			}
			current = append(current, currentPrices{optionID: get(r, "shipping_option_id"), priceSetID: get(r, "price_set_id"), prices: prices})
		}
		svc := pricing(ctx.Container)
		for _, d := range data {
			if !d.Defined {
				continue
			}
			var cur *currentPrices
			for i := range current {
				if fmt.Sprint(current[i].optionID) == fmt.Sprint(d.ID) {
					cur = &current[i]
				}
			}
			if cur == nil {
				continue
			}
			built := make([]any, len(d.Prices))
			for i, p := range d.Prices {
				extra := additionalRules(p)
				n := dal.NewRecord()
				for _, k := range p.Keys() {
					if k == "region_id" {
						continue
					}
					v, _ := p.Get(k)
					n.Set(k, v)
				}
				n.Set("price_set_id", cur.priceSetID)
				if region, ok := p.Get("region_id"); ok {
					n.Set("currency_code", currencies[fmt.Sprint(region)])
					rules := dal.NewRecord()
					rules.Set("region_id", region)
					for _, k := range extra.Keys() {
						v, _ := extra.Get(k)
						rules.Set(k, v)
					}
					n.Set("rules", rules)
				} else if _, ok := p.Get("currency_code"); ok {
					n.Set("rules", extra)
				}
				built[i] = n
			}
			upd := dal.NewRecord()
			upd.Set("prices", built)
			if _, err := svc.UpdatePriceSets(ctx.Context, cur.priceSetID, upd); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil, any(current)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		cur, _ := data.([]currentPrices)
		svc := pricing(ctx.Container)
		for _, c := range cur {
			if c.prices == nil {
				continue
			}
			upd := dal.NewRecord()
			upd.Set("prices", c.prices)
			if _, err := svc.UpdatePriceSets(ctx.Context, c.priceSetID, upd); err != nil {
				return err
			}
		}
		return nil
	})

// DeleteShippingOptionsStep mirrors deleteShippingOptionsStep.
var DeleteShippingOptionsStep = composer.CreateStep("delete-shipping-options-step",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[*dal.Record], error) {
		if len(ids) == 0 {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		res, err := Fulfillment(ctx.Container).Call(ctx.Context, "softDeleteShippingOptions", []any{ids})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		out := dal.NewRecord()
		out.Set(utilsmodulessdk.FULFILLMENT, res)
		return composer.NewStepResponse(out, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := Fulfillment(ctx.Container).Internal("ShippingOption").Restore(ctx.Context, ids)
		return err
	})
