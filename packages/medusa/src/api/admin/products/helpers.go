package products

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func isPricing(f string) bool {
	return strings.HasPrefix(f, "variants.prices") || strings.HasPrefix(f, "*variants.prices") || strings.HasPrefix(f, "prices") || strings.HasPrefix(f, "*prices")
}

// RemapKeysForProduct mirrors remapKeysForProduct.
func RemapKeysForProduct(fields []string) []string {
	var product, pricing []string
	for _, f := range fields {
		if isPricing(f) {
			pricing = append(pricing, strings.Replace(f, "variants.prices.", "variants.price_set.prices.", 1))
		} else {
			product = append(product, f)
		}
	}
	return append(product, pricing...)
}

// RemapKeysForVariant mirrors remapKeysForVariant.
func RemapKeysForVariant(fields []string) []string {
	var variant, pricing []string
	for _, f := range fields {
		if isPricing(f) {
			pricing = append(pricing, strings.Replace(f, "prices.", "price_set.prices.", 1))
		} else {
			variant = append(variant, f)
		}
	}
	return append(variant, pricing...)
}

func records(v any) ([]any, bool) {
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

// RemapProductResponse mirrors remapProductResponse.
func RemapProductResponse(p any) any {
	r, ok := p.(*dal.Record)
	if !ok || r == nil {
		return p
	}
	out := dal.NewRecord()
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	if v, ok := r.Get("variants"); ok {
		if list, ok := records(v); ok {
			mapped := make([]any, len(list))
			for i, x := range list {
				mapped[i] = RemapVariantResponse(x)
			}
			out.Set("variants", mapped)
		} else {
			// product.variants?.map(...) on null: undefined, dropped by JSON
			out.Delete("variants")
		}
	}
	return out
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
				if list, ok := records(pv); ok {
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
	list, _ := records(pr)
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
