package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// UpdatePriceSets mirrors updatePriceSets(idOrSelector, data).
func (s *PricingModuleService) UpdatePriceSets(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &PricingModuleService{MedusaService: tx}
		var input []*dal.Record
		if id, ok := idOrSelector.(string); ok {
			if _, err := t.Internal("PriceSet").Retrieve(ctx, id, utilsmodulessdk.FindConfig{}); err != nil {
				return err
			}
			input = []*dal.Record{withID(id, data)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			rows, err := t.Internal("PriceSet").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, r := range rows {
				id, _ := r.Get("id")
				input = append(input, withID(id, data))
			}
		}
		var err error
		out, err = t.updatePriceSets(ctx, input)
		return err
	})
	return out, err
}

// UpsertPriceSets mirrors upsertPriceSets.
func (s *PricingModuleService) UpsertPriceSets(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var forCreate, forUpdate []*dal.Record
	for _, p := range data {
		if id, _ := p.Get("id"); id != nil && id != "" {
			forUpdate = append(forUpdate, p)
		} else {
			forCreate = append(forCreate, p)
		}
	}
	var out []*dal.Record
	if len(forCreate) > 0 {
		created, err := s.CreatePriceSets(ctx, forCreate)
		if err != nil {
			return nil, err
		}
		out = append(out, created...)
	}
	if len(forUpdate) > 0 {
		err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
			t := &PricingModuleService{MedusaService: tx}
			updated, err := t.updatePriceSets(ctx, forUpdate)
			out = append(out, updated...)
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func withID(id any, data *dal.Record) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	if data != nil {
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			r.Set(k, v)
		}
	}
	return r
}

func listOf(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// updatePriceSets mirrors updatePriceSets_: the given prices replace the
// price set's prices (without price list); existing ones are updated by id.
func (s *PricingModuleService) updatePriceSets(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var ids []any
	var prices []*dal.Record
	for _, ps := range data {
		id, _ := ps.Get("id")
		ids = append(ids, id)
		raw, _ := ps.Get("prices")
		var withSet []any
		for _, p := range listOf(raw) {
			c := dal.NewRecord()
			for _, k := range p.Keys() {
				v, _ := p.Get(k)
				c.Set(k, v)
			}
			c.Set("price_set_id", id)
			withSet = append(withSet, c)
		}
		normalized, err := normalizePrices(withSet, nil)
		if err != nil {
			return nil, err
		}
		for _, n := range normalized {
			r := n.(*dal.Record)
			r.Delete("price_list_id")
			prices = append(prices, r)
		}
	}
	f := dal.NewRecord()
	f.Set("price_set_id", ids)
	f.Set("price_list_id", nil)
	existing, err := s.Internal("Price").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"price_rules"}})
	if err != nil {
		return nil, err
	}
	existingByID := map[string]*dal.Record{}
	for _, p := range existing {
		existingByID[fmt.Sprint(get(p, "id"))] = p
	}
	incoming := map[string]bool{}
	var toCreate []*dal.Record
	var toUpdate []*dal.Record
	for _, p := range prices {
		id, _ := get(p, "id").(string)
		if id != "" {
			incoming[id] = true
		}
		if id == "" || existingByID[id] == nil {
			if !p.Has("price_rules") {
				p.Set("price_rules", []any{})
			}
			toCreate = append(toCreate, p)
		} else {
			toUpdate = append(toUpdate, p)
		}
	}
	var toDelete []any
	for _, p := range existing {
		if !incoming[fmt.Sprint(get(p, "id"))] {
			toDelete = append(toDelete, get(p, "id"))
		}
	}
	if len(toCreate) > 0 {
		if _, err := s.Internal("Price").Create(ctx, toCreate); err != nil {
			return nil, err
		}
	}
	if len(toUpdate) > 0 {
		var updates []any
		for _, p := range toUpdate {
			id := fmt.Sprint(get(p, "id"))
			ex := existingByID[id]
			rulesIn, hasRules := p.Get("price_rules")
			rules := listOf(rulesIn)
			switch {
			case len(rules) > 0:
				existingRules := listOf(get(ex, "price_rules"))
				var create []*dal.Record
				var update []any
				keep := map[string]bool{}
				for _, r := range rules {
					if rid, ok := get(r, "id").(string); ok && rid != "" {
						keep[rid] = true
						update = append(update, r)
					} else {
						c := dal.NewRecord()
						for _, k := range r.Keys() {
							v, _ := r.Get(k)
							c.Set(k, v)
						}
						c.Set("price_id", id)
						create = append(create, c)
					}
				}
				var del []any
				for _, er := range existingRules {
					if !keep[fmt.Sprint(get(er, "id"))] {
						del = append(del, get(er, "id"))
					}
				}
				n := 0
				if len(create) > 0 {
					created, err := s.Internal("PriceRule").Create(ctx, create)
					if err != nil {
						return nil, err
					}
					n += len(created)
				}
				if len(update) > 0 {
					if _, err := s.Internal("PriceRule").Update(ctx, update); err != nil {
						return nil, err
					}
					n += len(update)
				}
				if len(del) > 0 {
					if _, err := s.Internal("PriceRule").Delete(ctx, del); err != nil {
						return nil, err
					}
				}
				p.Set("rules_count", float64(n))
			case hasRules && rulesIn != nil:
				var del []any
				for _, er := range listOf(get(ex, "price_rules")) {
					del = append(del, get(er, "id"))
				}
				if len(del) > 0 {
					if _, err := s.Internal("PriceRule").Delete(ctx, del); err != nil {
						return nil, err
					}
				}
				p.Set("rules_count", float64(0))
			default:
				p.Delete("rules_count")
			}
			p.Delete("price_rules")
			updates = append(updates, p)
		}
		if _, err := s.Internal("Price").Update(ctx, updates); err != nil {
			return nil, err
		}
	}
	if len(toDelete) > 0 {
		if _, err := s.Internal("Price").Delete(ctx, toDelete); err != nil {
			return nil, err
		}
	}
	sf := dal.NewRecord()
	sf.Set("id", ids)
	return s.Internal("PriceSet").List(ctx, sf, normalizePriceSetConfig(utilsmodulessdk.FindConfig{Relations: []string{"prices", "prices.price_rules"}}))
}

// CreatePricePreferences mirrors createPricePreferences (is_tax_inclusive
// defaults to false).
func (s *PricingModuleService) CreatePricePreferences(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	in := make([]*dal.Record, len(data))
	for i, d := range data {
		r := dal.NewRecord()
		for _, k := range d.Keys() {
			v, _ := d.Get(k)
			r.Set(k, v)
		}
		if v, _ := r.Get("is_tax_inclusive"); v == nil {
			r.Set("is_tax_inclusive", false)
		}
		in[i] = r
	}
	return s.Internal("PricePreference").Create(ctx, in)
}

// UpdatePricePreferences mirrors updatePricePreferences(idOrSelector, data).
func (s *PricingModuleService) UpdatePricePreferences(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var input []any
	if id, ok := idOrSelector.(string); ok {
		if _, err := s.Internal("PricePreference").Retrieve(ctx, id, utilsmodulessdk.FindConfig{}); err != nil {
			return nil, err
		}
		input = []any{withID(id, data)}
	} else {
		sel, _ := idOrSelector.(*dal.Record)
		rows, err := s.Internal("PricePreference").List(ctx, sel, utilsmodulessdk.FindConfig{})
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			id, _ := r.Get("id")
			input = append(input, withID(id, data))
		}
	}
	if len(input) == 0 {
		return []*dal.Record{}, nil
	}
	res, err := s.Internal("PricePreference").Update(ctx, input)
	if err != nil {
		return nil, err
	}
	return listOf(res), nil
}

// UpsertPricePreferences mirrors upsertPricePreferences.
func (s *PricingModuleService) UpsertPricePreferences(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var forCreate []*dal.Record
	var forUpdate []any
	for _, d := range data {
		if id, _ := d.Get("id"); id != nil && id != "" {
			forUpdate = append(forUpdate, d)
		} else {
			forCreate = append(forCreate, d)
		}
	}
	var out []*dal.Record
	if len(forCreate) > 0 {
		c, err := s.CreatePricePreferences(ctx, forCreate)
		if err != nil {
			return nil, err
		}
		out = append(out, c...)
	}
	if len(forUpdate) > 0 {
		res, err := s.Internal("PricePreference").Update(ctx, forUpdate)
		if err != nil {
			return nil, err
		}
		out = append(out, listOf(res)...)
	}
	return out, nil
}
