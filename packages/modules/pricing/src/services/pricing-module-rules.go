package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// AddPrices mirrors addPrices: the prices ({ priceSetId, prices }) are
// normalized against the existing ones (matched by id or by their
// constraints) and upserted with their rules; the price sets are returned
// with their prices, in input order.
func (s *PricingModuleService) AddPrices(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		var setIDs []any
		var prices []*dal.Record
		for _, d := range data {
			setIDs = append(setIDs, get(d, "priceSetId"))
			for _, p := range listOf(get(d, "prices")) {
				p.Set("price_set_id", get(d, "priceSetId"))
				prices = append(prices, p)
			}
		}
		f := dal.NewRecord()
		f.Set("id", setIDs)
		sets, err := tx.Internal("PriceSet").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		existing, err := tx.Internal("Price").List(ctx, priceConstraints(prices, nil), utilsmodulessdk.FindConfig{Relations: []string{"price_rules"}})
		if err != nil {
			return err
		}
		var toUpsert []*dal.Record
		for _, d := range data {
			var in []any
			for _, p := range listOf(get(d, "prices")) {
				in = append(in, p)
			}
			np, err := normalizePricesWith(in, existing, nil)
			if err != nil {
				return err
			}
			for _, p := range np {
				if r, ok := p.(*dal.Record); ok {
					toUpsert = append(toUpsert, r)
				}
			}
		}
		known := map[string]bool{}
		for _, ps := range sets {
			known[fmt.Sprint(get(ps, "id"))] = true
		}
		for _, p := range toUpsert {
			if id := get(p, "price_set_id"); !known[fmt.Sprint(id)] {
				return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Price set with id: %v not found", id))
			}
		}
		_, err = tx.Internal("Price").UpsertWithReplace(ctx, toUpsert, []string{"price_rules"})
		return err
	})
	if err != nil {
		return nil, err
	}
	var ids []any
	for _, d := range data {
		ids = append(ids, get(d, "priceSetId"))
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	sets, err := s.ListBySuffix(ctx, "PriceSets", f, utilsmodulessdk.FindConfig{Relations: []string{"prices"}})
	if err != nil {
		return nil, err
	}
	byID := map[string]*dal.Record{}
	for _, ps := range sets {
		byID[fmt.Sprint(get(ps, "id"))] = ps
	}
	out := make([]*dal.Record, 0, len(data))
	for _, d := range data {
		out = append(out, byID[fmt.Sprint(get(d, "priceSetId"))])
	}
	return out, nil
}

// changePriceListRules is the shared body of setPriceListRules_ and
// removePriceListRules_: the rules of each price list are merged with the
// changes (attribute -> value, nil removes) and the price list upserted
// with its rules replaced and rules_count updated.
func (s *PricingModuleService) changePriceListRules(ctx context.Context, priceListID any, changes [][2]any, dropEmpty bool) (*dal.Record, error) {
	var out *dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		f := dal.NewRecord()
		f.Set("id", []any{priceListID})
		lists, err := tx.Internal("PriceList").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"price_list_rules"}})
		if err != nil {
			return err
		}
		if len(lists) == 0 || len(changes) == 0 {
			return nil
		}
		list := lists[0]
		var order []string
		values := map[string]any{}
		for _, r := range listOf(get(list, "price_list_rules")) {
			a := fmt.Sprint(get(r, "attribute"))
			if _, ok := values[a]; !ok {
				order = append(order, a)
			}
			values[a] = get(r, "value")
		}
		for _, c := range changes {
			a := fmt.Sprint(c[0])
			if _, ok := values[a]; !ok {
				order = append(order, a)
			}
			values[a] = c[1]
		}
		rules := []any{}
		for _, a := range order {
			v := values[a]
			if dropEmpty && !jsTruthy(v) {
				continue
			}
			r := dal.NewRecord()
			r.Set("attribute", a)
			r.Set("value", v)
			rules = append(rules, r)
		}
		// { ...priceList }: the entity's properties, its primary key after the
		// hydrated columns (MikroORM entity property order)
		up := dal.NewRecord()
		for _, k := range list.Keys() {
			if k == "price_list_rules" || k == "id" {
				continue
			}
			v, _ := list.Get(k)
			up.Set(k, v)
		}
		up.Set("id", get(list, "id"))
		// Map.size: the attributes, including removed ones (upstream counts
		// the merged map before filtering)
		up.Set("rules_count", float64(len(order)))
		up.Set("price_list_rules", rules)
		res, err := tx.Internal("PriceList").UpsertWithReplace(ctx, []*dal.Record{up}, []string{"price_list_rules"})
		if err != nil {
			return err
		}
		if len(res) > 0 {
			out = res[0]
		}
		return nil
	})
	return out, err
}

// SetPriceListRules mirrors setPriceListRules ({ price_list_id, rules:
// { attribute: value } }).
func (s *PricingModuleService) SetPriceListRules(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	var changes [][2]any
	if rules, ok := get(data, "rules").(*dal.Record); ok {
		for _, k := range rules.Keys() {
			v, _ := rules.Get(k)
			changes = append(changes, [2]any{k, v})
		}
	}
	return s.changePriceListRules(ctx, get(data, "price_list_id"), changes, false)
}

// RemovePriceListRules mirrors removePriceListRules ({ price_list_id,
// rules: [attribute] }).
func (s *PricingModuleService) RemovePriceListRules(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	var changes [][2]any
	if rules, ok := get(data, "rules").([]any); ok {
		for _, a := range rules {
			changes = append(changes, [2]any{a, nil})
		}
	}
	return s.changePriceListRules(ctx, get(data, "price_list_id"), changes, true)
}
