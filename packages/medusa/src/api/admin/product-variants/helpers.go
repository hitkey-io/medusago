package productvariants

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// These helpers are defined in api/admin/products/helpers.ts upstream; they
// live here because Go packages cannot import each other (products imports
// the product-variants validators). products re-exports them.

// IsPricing reports whether a field selects prices.
func IsPricing(f string) bool {
	return strings.HasPrefix(f, "variants.prices") || strings.HasPrefix(f, "*variants.prices") || strings.HasPrefix(f, "prices") || strings.HasPrefix(f, "*prices")
}

// Records returns a list of records as []any.
func Records(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out, true
	}
	return nil, false
}

// RemapKeysForVariant mirrors remapKeysForVariant.
func RemapKeysForVariant(fields []string) []string {
	var variant, pricing []string
	for _, f := range fields {
		if IsPricing(f) {
			pricing = append(pricing, strings.Replace(f, "prices.", "price_set.prices.", 1))
		} else {
			variant = append(variant, f)
		}
	}
	return append(variant, pricing...)
}

// RemapVariantResponse mirrors remapVariantResponse.
func RemapVariantResponse(v any) any {
	variant, ok := v.(*dal.Record)
	if !ok || variant == nil {
		return v
	}
	resp := dal.NewRecord()
	for _, k := range variant.Keys() {
		x, _ := variant.Get(k)
		resp.Set(k, x)
	}
	var prices []any
	hasPrices := false
	if ps, ok := variant.Get("price_set"); ok {
		if psr, ok := ps.(*dal.Record); ok && psr != nil {
			if pv, ok := psr.Get("prices"); ok {
				if list, ok := Records(pv); ok {
					hasPrices = true
					id, _ := variant.Get("id")
					for _, p := range list {
						price, _ := p.(*dal.Record)
						if price == nil {
							continue
						}
						out := dal.NewRecord()
						for _, k := range []string{"id", "amount", "currency_code", "min_quantity", "max_quantity"} {
							x, _ := price.Get(k)
							if x != nil || price.Has(k) {
								out.Set(k, x)
							}
						}
						out.Set("variant_id", id)
						for _, k := range []string{"created_at", "updated_at"} {
							x, _ := price.Get(k)
							if x != nil || price.Has(k) {
								out.Set(k, x)
							}
						}
						out.Set("rules", buildRules(price))
						prices = append(prices, out)
					}
				}
			}
		}
	}
	if hasPrices {
		if prices == nil {
			prices = []any{}
		}
		resp.Set("prices", prices)
	} else {
		resp.Set("prices", undefinedMarker)
	}
	resp.Delete("price_set")
	if p, _ := resp.Get("prices"); p == undefinedMarker {
		resp.Delete("prices")
	}
	return resp
}

type undefinedT struct{}

var undefinedMarker = &undefinedT{}

func buildRules(price *dal.Record) *dal.Record {
	rules := dal.NewRecord()
	pr, _ := price.Get("price_rules")
	list, _ := Records(pr)
	for _, x := range list {
		rule, _ := x.(*dal.Record)
		if rule == nil {
			continue
		}
		attr, _ := rule.Get("attribute")
		if a, ok := attr.(string); ok && a != "" {
			val, _ := rule.Get("value")
			rules.Set(a, val)
		}
	}
	return rules
}
