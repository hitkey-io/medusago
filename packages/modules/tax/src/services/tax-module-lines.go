package services

import (
	"context"
	"fmt"
	taxtypes "github.com/hitkey-io/medusago/packages/core/types/src/tax"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

func rget(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func rstr(r *dal.Record, k string) string {
	if v := rget(r, k); v != nil {
		return fmt.Sprint(v)
	}
	return ""
}

func rrecords(v any) []*dal.Record {
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

func isShippingItem(item *dal.Record) bool { return item.Has("shipping_option_id") }

type ruleQuery struct{ reference, referenceID string }

func itemRuleQueries(item *dal.Record) []ruleQuery {
	if isShippingItem(item) {
		return []ruleQuery{{"shipping_option", rstr(item, "shipping_option_id")}}
	}
	return []ruleQuery{{"product", rstr(item, "product_id")}, {"product_type", rstr(item, "product_type_id")}}
}

// rateMatchesQuery mirrors rateMatchesQuery.
func rateMatchesQuery(rate *dal.Record, regionIDs map[string]bool, rules []ruleQuery) bool {
	if !regionIDs[rstr(rate, "tax_region_id")] {
		return false
	}
	if b, _ := rget(rate, "is_default").(bool); b {
		return true
	}
	for _, q := range rules {
		for _, r := range rrecords(rget(rate, "rules")) {
			if rstr(r, "reference") == q.reference && rstr(r, "reference_id") == q.referenceID && q.referenceID != "" {
				return true
			}
		}
	}
	return false
}

// prioritizeRates mirrors prioritizeRates (a stable sort by priority score).
func prioritizeRates(rates []*dal.Record, item *dal.Record) []*dal.Record {
	type scored struct {
		rate  *dal.Record
		score int
	}
	list := make([]scored, len(rates))
	shipping := isShippingItem(item)
	for i, rate := range rates {
		rules := rrecords(rget(rate, "rules"))
		productMatch, typeMatch, shippingMatch := false, false, false
		if len(rules) > 0 {
			var matching []*dal.Record
			for _, r := range rules {
				ref, refID := rstr(r, "reference"), rstr(r, "reference_id")
				if shipping {
					if ref == "shipping_option" && refID == rstr(item, "shipping_option_id") {
						matching = append(matching, r)
					}
				} else if (ref == "product" && refID == rstr(item, "product_id")) || (ref == "product_type" && refID == rstr(item, "product_type_id")) {
					matching = append(matching, r)
				}
			}
			for _, m := range matching {
				if rstr(m, "reference") == "product" {
					productMatch = true
				}
			}
			typeMatch = len(matching) > 0
			shippingMatch = shipping && len(matching) > 0
		}
		region, _ := rget(rate, "tax_region").(*dal.Record)
		isProvince := rget(region, "province_code") != nil
		isDefault, _ := rget(rate, "is_default").(bool)
		score := 7
		switch {
		case (shippingMatch || productMatch) && isProvince:
			score = 1
		case typeMatch && isProvince:
			score = 2
		case isDefault && isProvince:
			score = 3
		case (shippingMatch || productMatch) && !isProvince:
			score = 4
		case typeMatch && !isProvince:
			score = 5
		case isDefault && !isProvince:
			score = 6
		}
		list[i] = scored{rate, score}
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].score < list[j].score })
	out := make([]*dal.Record, len(list))
	for i, s := range list {
		out[i] = s.rate
	}
	return out
}

// getTaxRatesForItem mirrors getTaxRatesForItem.
func getTaxRatesForItem(item *dal.Record, rates []*dal.Record) []*dal.Record {
	if len(rates) == 0 {
		return nil
	}
	prioritized := prioritizeRates(rates, item)
	rate := prioritized[0]
	out := []*dal.Record{rate}
	region, _ := rget(rate, "tax_region").(*dal.Record)
	combinable, _ := rget(rate, "is_combinable").(bool)
	parentID := rstr(region, "parent_id")
	if !combinable || parentID == "" {
		return out
	}
	for _, r := range prioritized {
		if reg, _ := rget(r, "tax_region").(*dal.Record); reg != nil && rstr(reg, "id") == parentID {
			out = append(out, r)
			break
		}
	}
	return out
}

// GetTaxLines mirrors getTaxLines with the system tax provider: items are
// { id, product_id, product_type_id } or { id, shipping_option_id };
// calculationContext carries address.{country_code, province_code}.
func (s *TaxModuleService) GetTaxLines(ctx context.Context, items []*dal.Record, calculationContext *dal.Record) ([]*dal.Record, error) {
	address, _ := rget(calculationContext, "address").(*dal.Record)
	country := strings.ToLower(rstr(address, "country_code"))
	var province any
	if p := rstr(address, "province_code"); p != "" {
		province = strings.ToLower(p)
	}
	a := dal.NewRecord()
	a.Set("country_code", country)
	a.Set("province_code", nil)
	b := dal.NewRecord()
	b.Set("country_code", country)
	b.Set("province_code", province)
	f := dal.NewRecord()
	f.Set("$or", []any{a, b})
	regions, err := s.Internal("TaxRegion").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	var parent *dal.Record
	regionIDs := map[string]bool{}
	var ids []any
	for _, r := range regions {
		if parent == nil && rget(r, "province_code") == nil {
			parent = r
		}
		regionIDs[rstr(r, "id")] = true
		ids = append(ids, rget(r, "id"))
	}
	if parent == nil {
		return []*dal.Record{}, nil
	}
	rf := dal.NewRecord()
	rf.Set("tax_region_id", ids)
	candidates, err := s.Internal("TaxRate").List(ctx, rf, utilsmodulessdk.FindConfig{Relations: []string{"tax_region", "rules"}})
	if err != nil {
		return nil, err
	}
	var itemLines []taxtypes.ItemTaxCalculationLine
	var shippingLines []taxtypes.ShippingTaxCalculationLine
	for _, item := range items {
		queries := itemRuleQueries(item)
		var matched []*dal.Record
		for _, rate := range candidates {
			if rateMatchesQuery(rate, regionIDs, queries) {
				matched = append(matched, rate)
			}
		}
		rates := getTaxRatesForItem(item, matched)
		if isShippingItem(item) {
			shippingLines = append(shippingLines, taxtypes.ShippingTaxCalculationLine{ShippingLine: item, Rates: rates})
		} else {
			itemLines = append(itemLines, taxtypes.ItemTaxCalculationLine{LineItem: item, Rates: rates})
		}
	}
	// getTaxLinesFromProvider: the provider of the parent region
	providerID := fmt.Sprint(rget(parent, "provider_id"))
	if rget(parent, "provider_id") == nil {
		providerID = "null"
	}
	provider, err := s.retrieveProvider(providerID)
	if err != nil {
		return nil, err
	}
	out, err := provider.GetTaxLines(ctx, itemLines, shippingLines, calculationContext)
	if err != nil {
		return nil, err
	}
	if out == nil {
		out = []*dal.Record{}
	}
	return out, nil
}
