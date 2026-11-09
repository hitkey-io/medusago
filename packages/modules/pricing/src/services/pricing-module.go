// Package services mirrors @medusajs/pricing/src/services.
package services

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/pricing/src/models"
)

// PricingModuleService mirrors PricingModuleService.
type PricingModuleService struct {
	*utilsmodulessdk.MedusaService
}

// NewPricingModuleService builds the service.
func NewPricingModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*PricingModuleService, error) {
	return &PricingModuleService{MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig)}, nil
}

// normalizePriceSetConfig mirrors normalizePriceSetConfig: the prices of
// price lists are not populated with the price sets.
func normalizePriceSetConfig(config utilsmodulessdk.FindConfig) utilsmodulessdk.FindConfig {
	if config.PopulateWhere == nil {
		pl := dal.NewRecord()
		pl.Set("price_list_id", nil)
		config.PopulateWhere = map[string]*dal.Record{"prices": pl}
	}
	return config
}

// setupCalculatedPriceConfig mirrors setupCalculatedPriceConfig_: the
// pricing context of the filters, when the virtual calculated_price
// relation is requested (removed from the config).
func setupCalculatedPriceConfig(filters *dal.Record, config utilsmodulessdk.FindConfig) (*dal.Record, utilsmodulessdk.FindConfig, *dal.Record, bool) {
	var pricingContext *dal.Record
	if filters != nil && filters.Has("context") {
		v, _ := filters.Get("context")
		pricingContext, _ = v.(*dal.Record)
		out := dal.NewRecord()
		for _, k := range filters.Keys() {
			if k == "context" {
				continue
			}
			x, _ := filters.Get(k)
			out.Set(k, x)
		}
		filters = out
	}
	idx := slices.Index(config.Relations, "calculated_price")
	if idx < 0 {
		return filters, config, nil, false
	}
	config.Relations = append(append([]string{}, config.Relations[:idx]...), config.Relations[idx+1:]...)
	if config.Select != nil {
		var sel []string
		for _, f := range config.Select {
			if f == "calculated_price" || strings.HasPrefix(f, "calculated_price.") {
				continue
			}
			sel = append(sel, f)
		}
		if sel == nil {
			sel = []string{}
		}
		config.Select = sel
	}
	if pricingContext == nil {
		pricingContext = dal.NewRecord()
	}
	return filters, config, pricingContext, true
}

func (s *PricingModuleService) attachCalculatedPrices(ctx context.Context, sets []*dal.Record, pricingContext *dal.Record) error {
	if len(sets) == 0 {
		return nil
	}
	ids := make([]any, len(sets))
	for i, ps := range sets {
		ids[i] = get(ps, "id")
	}
	prices, err := s.CalculatePrices(ctx, ids, pricingContext)
	if err != nil {
		return err
	}
	byID := map[string]*dal.Record{}
	for _, p := range prices {
		byID[fmt.Sprint(get(p, "id"))] = p
	}
	for _, ps := range sets {
		if p := byID[fmt.Sprint(get(ps, "id"))]; p != nil {
			ps.Set("calculated_price", p)
		} else {
			ps.Set("calculated_price", nil)
		}
	}
	return nil
}

// ListBySuffix dispatches list<suffix> to the overridden methods.
func (s *PricingModuleService) ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	if suffix == "PriceSets" {
		config = normalizePriceSetConfig(config)
		var pricingContext *dal.Record
		var calc bool
		filters, config, pricingContext, calc = setupCalculatedPriceConfig(filters, config)
		sets, err := s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
		if err != nil || !calc {
			return sets, err
		}
		return sets, s.attachCalculatedPrices(ctx, sets, pricingContext)
	}
	return s.MedusaService.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCountBySuffix dispatches listAndCount<suffix>.
func (s *PricingModuleService) ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	if suffix == "PriceSets" {
		config = normalizePriceSetConfig(config)
		var pricingContext *dal.Record
		var calc bool
		filters, config, pricingContext, calc = setupCalculatedPriceConfig(filters, config)
		sets, count, err := s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
		if err != nil || !calc {
			return sets, count, err
		}
		return sets, count, s.attachCalculatedPrices(ctx, sets, pricingContext)
	}
	return s.MedusaService.ListAndCountBySuffix(ctx, suffix, filters, config)
}

var ruleOperators = []string{"gte", "lte", "gt", "lt", "eq"}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%f", t), "0"), ".")
	}
	return fmt.Sprint(v)
}

// hashPrice mirrors hashPrice.
func hashPrice(p *dal.Record) string {
	var parts []string
	if p.Has("currency_code") {
		parts = append(parts, "cc:"+strings.ToLower(jsString(get(p, "currency_code"))))
	}
	if p.Has("price_set_id") {
		parts = append(parts, "ps:"+jsString(get(p, "price_set_id")))
	}
	if p.Has("price_list_id") {
		parts = append(parts, "pl:"+jsString(get(p, "price_list_id")))
	}
	if p.Has("min_quantity") {
		parts = append(parts, "min:"+jsString(get(p, "min_quantity")))
	}
	if p.Has("max_quantity") {
		parts = append(parts, "max:"+jsString(get(p, "max_quantity")))
	}
	if rules := listOf(get(p, "price_rules")); len(rules) > 0 {
		var rs []string
		for _, rr := range rules {
			rs = append(rs, jsString(get(rr, "attribute"))+"="+jsString(get(rr, "value")))
		}
		parts = append(parts, "rules:"+strings.Join(rs, ","))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

// normalizePrices mirrors normalizePrices without existing prices.
func normalizePrices(data []any, priceListID any) ([]any, error) {
	return normalizePricesWith(data, nil, priceListID)
}

// normalizePricesWith mirrors normalizePrices: an input price matching an
// existing one (same hash) takes its id and price rules.
func normalizePricesWith(data []any, existingPrices []*dal.Record, priceListID any) ([]any, error) {
	existingByHash := map[string]*dal.Record{}
	for _, p := range existingPrices {
		existingByHash[hashPrice(p)] = p
	}
	var order []string
	byKey := map[string]*dal.Record{}
	for _, d := range data {
		price, _ := d.(*dal.Record)
		if price == nil {
			continue
		}
		entry := dal.NewRecord()
		for _, k := range price.Keys() {
			v, _ := price.Get(k)
			entry.Set(k, v)
		}
		rulesIn, hasRules := price.Get("rules")
		var rules []any
		if rr, ok := rulesIn.(*dal.Record); ok {
			for _, attr := range rr.Keys() {
				v, _ := rr.Get(attr)
				if v == nil {
					continue
				}
				if arr, ok := v.([]any); ok {
					for _, c := range arr {
						cr, _ := c.(*dal.Record)
						op, _ := get(cr, "operator").(string)
						valid := false
						for _, o := range ruleOperators {
							if o == op {
								valid = true
							}
						}
						if !valid {
							return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "operator should be one of "+strings.Join(ruleOperators, ", "))
						}
						num, ok := get(cr, "value").(float64)
						if !ok {
							return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "value should be a number")
						}
						r := dal.NewRecord()
						r.Set("attribute", attr)
						r.Set("operator", op)
						r.Set("value", jsString(num))
						rules = append(rules, r)
					}
					continue
				}
				r := dal.NewRecord()
				r.Set("attribute", attr)
				r.Set("value", v)
				rules = append(rules, r)
			}
		}
		entry.Set("price_list_id", priceListID)
		if hasRules && rulesIn != nil {
			if rules == nil {
				rules = []any{}
			}
			entry.Set("price_rules", rules)
			entry.Set("rules_count", float64(len(rules)))
			entry.Delete("rules")
		}
		if cc, ok := entry.Get("currency_code"); ok && cc != nil {
			entry.Set("currency_code", strings.ToLower(fmt.Sprint(cc)))
		}
		key := hashPrice(entry)
		if existing := existingByHash[key]; existing != nil {
			if id := get(existing, "id"); id != nil {
				entry.Set("id", id)
			}
			if pr := get(existing, "price_rules"); pr != nil {
				entry.Set("price_rules", pr)
			}
		}
		if id, _ := get(entry, "id").(string); id != "" {
			key = id
		}
		if _, ok := byKey[key]; !ok {
			order = append(order, key)
		}
		byKey[key] = entry
	}
	out := make([]any, 0, len(order))
	for _, k := range order {
		out = append(out, byKey[k])
	}
	return out, nil
}

// CreatePriceSets mirrors createPriceSets: the price sets with their
// prices (and the rules of the prices), returned with prices and
// prices.price_rules.
func (s *PricingModuleService) CreatePriceSets(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	toCreate := make([]*dal.Record, len(data))
	for i, ps := range data {
		entry := dal.NewRecord()
		for _, k := range ps.Keys() {
			v, _ := ps.Get(k)
			entry.Set(k, v)
		}
		prices, _ := ps.Get("prices")
		pl, _ := prices.([]any)
		normalized, err := normalizePrices(pl, nil)
		if err != nil {
			return nil, err
		}
		entry.Set("prices", normalized)
		toCreate[i] = entry
	}
	created, err := s.Internal("PriceSet").Create(ctx, toCreate)
	if err != nil {
		return nil, err
	}
	ids := make([]any, len(created))
	for i, c := range created {
		ids[i], _ = c.Get("id")
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	rows, err := s.Internal("PriceSet").List(ctx, f, normalizePriceSetConfig(utilsmodulessdk.FindConfig{Relations: []string{"prices", "prices.price_rules"}}))
	if err != nil {
		return nil, err
	}
	byID := map[any]*dal.Record{}
	for _, r := range rows {
		id, _ := r.Get("id")
		byID[id] = r
	}
	out := make([]*dal.Record, len(created))
	for i, id := range ids {
		out[i] = byID[id]
	}
	return out, nil
}

// DeletePriceSets mirrors deletePriceSets (generated delete).
func (s *PricingModuleService) DeletePriceSets(ctx context.Context, ids []any) error {
	_, err := s.Internal("PriceSet").Delete(ctx, ids)
	return err
}
