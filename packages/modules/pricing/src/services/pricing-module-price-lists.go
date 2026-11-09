package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

func copyRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

// normalizePriceListDate mirrors normalizePriceListDate (with
// validatePriceListDates).
func normalizePriceListDate(data []*dal.Record) ([]*dal.Record, error) {
	out := make([]*dal.Record, len(data))
	for i, d := range data {
		pl := copyRecord(d)
		for _, k := range []string{"starts_at", "ends_at"} {
			v, ok := pl.Get(k)
			if !ok || !jsTruthy(v) {
				continue
			}
			if !common.IsDate(v) {
				label := "starts at"
				if k == "ends_at" {
					label = "ends at"
				}
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Cannot set price list %s with with invalid date string: %v", label, v))
			}
			pl.Set(k, common.GetIsoStringFromDate(v))
		}
		out[i] = pl
	}
	return out, nil
}

func jsTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0
	}
	return true
}

// priceListRules mirrors the rules -> price_list_rules conversion
// (removeNullish, deduplicated by JSON.stringify of the rule).
func priceListRules(rules *dal.Record) ([]any, int) {
	var out []any
	seen := map[string]bool{}
	count := 0
	for _, attr := range rules.Keys() {
		v, _ := rules.Get(attr)
		if v == nil {
			continue
		}
		count++
		r := dal.NewRecord()
		r.Set("attribute", attr)
		r.Set("value", v)
		b, _ := json.Marshal(r)
		if seen[string(b)] {
			continue
		}
		seen[string(b)] = true
		out = append(out, r)
	}
	if out == nil {
		out = []any{}
	}
	return out, count
}

// CreatePriceLists mirrors createPriceLists.
func (s *PricingModuleService) CreatePriceLists(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	normalized, err := normalizePriceListDate(data)
	if err != nil {
		return nil, err
	}
	toCreate := make([]*dal.Record, len(normalized))
	for i, pl := range normalized {
		entry := copyRecord(pl)
		entry.Delete("rules")
		if prices, ok := pl.Get("prices"); ok && prices != nil {
			list, _ := prices.([]any)
			np, err := normalizePrices(list, nil)
			if err != nil {
				return nil, err
			}
			for _, p := range np {
				p.(*dal.Record).Delete("price_list_id")
			}
			entry.Set("prices", np)
		}
		if rules, _ := pl.Get("rules"); rules != nil {
			rr, _ := rules.(*dal.Record)
			if rr == nil {
				rr = dal.NewRecord()
			}
			list, count := priceListRules(rr)
			entry.Set("price_list_rules", list)
			entry.Set("rules_count", float64(count))
		}
		toCreate[i] = entry
	}
	return s.Internal("PriceList").Create(ctx, toCreate)
}

func isPresentRecord(v any) bool {
	r, ok := v.(*dal.Record)
	return ok && r != nil && len(r.Keys()) > 0
}

func mergeMetadata(current any, update *dal.Record) *dal.Record {
	merged := dal.NewRecord()
	if cur, ok := current.(*dal.Record); ok {
		for _, k := range cur.Keys() {
			v, _ := cur.Get(k)
			merged.Set(k, v)
		}
	}
	for _, k := range update.Keys() {
		v, _ := update.Get(k)
		if v == "" {
			merged.Delete(k)
			continue
		}
		merged.Set(k, v)
	}
	return merged
}

func missingIDs(want []any, found []*dal.Record) []string {
	have := map[string]bool{}
	for _, f := range found {
		have[fmt.Sprint(get(f, "id"))] = true
	}
	var out []string
	for _, w := range want {
		if !have[fmt.Sprint(w)] {
			out = append(out, fmt.Sprint(w))
		}
	}
	return out
}

// UpdatePriceLists mirrors updatePriceLists.
func (s *PricingModuleService) UpdatePriceLists(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		ids := make([]any, len(data))
		for i, d := range data {
			ids[i] = get(d, "id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		existing, err := tx.Internal("PriceList").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if len(existing) != len(data) {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Price lists with ids: '%s' not found", strings.Join(missingIDs(ids, existing), ", ")))
		}
		byID := map[string]*dal.Record{}
		for _, e := range existing {
			byID[fmt.Sprint(get(e, "id"))] = e
		}
		normalized, err := normalizePriceListDate(data)
		if err != nil {
			return err
		}
		entries := make([]*dal.Record, len(normalized))
		for i, pl := range normalized {
			entry := copyRecord(pl)
			entry.Delete("rules")
			entry.Delete("price_list_rules")
			if md, _ := pl.Get("metadata"); isPresentRecord(md) {
				entry.Set("metadata", mergeMetadata(get(byID[fmt.Sprint(get(pl, "id"))], "metadata"), md.(*dal.Record)))
			}
			if rules, ok := pl.Get("rules"); ok {
				rr, _ := rules.(*dal.Record)
				if rr != nil || rules == nil {
					if rr == nil {
						rr = dal.NewRecord()
					}
					list, count := priceListRules(rr)
					entry.Set("price_list_rules", list)
					entry.Set("rules_count", float64(count))
				}
			}
			entries[i] = entry
		}
		out, err = tx.Internal("PriceList").UpsertWithReplace(ctx, entries, []string{"price_list_rules"})
		return err
	})
	return out, err
}

// priceConstraints mirrors buildPreNormalizationPriceConstraintsFromData.
func priceConstraints(prices []*dal.Record, priceListIDs []any) *dal.Record {
	var withIDs []any
	var constraints []any
	seen := map[string]bool{}
	for _, p := range prices {
		if id := get(p, "id"); jsTruthy(id) {
			withIDs = append(withIDs, id)
			continue
		}
		c := dal.NewRecord()
		if cc := get(p, "currency_code"); jsTruthy(cc) {
			c.Set("currency_code", strings.ToLower(fmt.Sprint(cc)))
		}
		if v, ok := p.Get("price_set_id"); ok {
			c.Set("price_set_id", v)
		}
		if v, ok := p.Get("price_list_id"); ok {
			c.Set("price_list_id", v)
		}
		if v, ok := p.Get("min_quantity"); ok {
			c.Set("min_quantity", v)
		}
		if v, ok := p.Get("max_quantity"); ok {
			c.Set("max_quantity", v)
		}
		b, _ := json.Marshal(c)
		if !seen[string(b)] {
			seen[string(b)] = true
			constraints = append(constraints, c)
		}
	}
	q := dal.NewRecord()
	if len(priceListIDs) > 0 {
		in := dal.NewRecord()
		in.Set("$in", priceListIDs)
		q.Set("price_list_id", in)
	}
	switch {
	case len(withIDs) > 0 && len(constraints) > 0:
		idf := dal.NewRecord()
		idf.Set("id", withIDs)
		q.Set("$or", append([]any{idf}, constraints...))
	case len(withIDs) > 0:
		q.Set("id", withIDs)
	case len(constraints) == 1:
		c := constraints[0].(*dal.Record)
		for _, k := range c.Keys() {
			v, _ := c.Get(k)
			q.Set(k, v)
		}
	case len(constraints) > 1:
		q.Set("$or", constraints)
	}
	return q
}

// upsertPriceListPrices is the shared body of addPriceListPrices_ and
// updatePriceListPrices_.
func (s *PricingModuleService) upsertPriceListPrices(ctx context.Context, data []*dal.Record, checkEach bool) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		var listIDs []any
		var prices []*dal.Record
		for _, d := range data {
			listIDs = append(listIDs, get(d, "price_list_id"))
			prices = append(prices, listOf(get(d, "prices"))...)
		}
		f := dal.NewRecord()
		f.Set("id", listIDs)
		lists, err := tx.Internal("PriceList").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		existing, err := tx.Internal("Price").List(ctx, priceConstraints(prices, listIDs), utilsmodulessdk.FindConfig{Relations: []string{"price_rules"}})
		if err != nil {
			return err
		}
		var toUpsert []*dal.Record
		for _, d := range data {
			var in []any
			for _, p := range listOf(get(d, "prices")) {
				in = append(in, copyRecord(p))
			}
			np, err := normalizePricesWith(in, existing, get(d, "price_list_id"))
			if err != nil {
				return err
			}
			for _, p := range np {
				toUpsert = append(toUpsert, p.(*dal.Record))
			}
		}
		known := map[string]bool{}
		for _, l := range lists {
			known[fmt.Sprint(get(l, "id"))] = true
		}
		if checkEach {
			for _, d := range data {
				if id := get(d, "price_list_id"); !known[fmt.Sprint(id)] {
					return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Price list with id: %v not found", id))
				}
			}
		} else {
			for _, p := range toUpsert {
				if id := get(p, "price_list_id"); !known[fmt.Sprint(id)] {
					return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Price list with id: %v not found", id))
				}
			}
		}
		out, err = tx.Internal("Price").UpsertWithReplace(ctx, toUpsert, []string{"price_rules"})
		return err
	})
	return out, err
}

// AddPriceListPrices mirrors addPriceListPrices.
func (s *PricingModuleService) AddPriceListPrices(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.upsertPriceListPrices(ctx, data, false)
}

// UpdatePriceListPrices mirrors updatePriceListPrices.
func (s *PricingModuleService) UpdatePriceListPrices(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.upsertPriceListPrices(ctx, data, true)
}

// RemovePrices mirrors removePrices (hard delete).
func (s *PricingModuleService) RemovePrices(ctx context.Context, ids []any) error {
	_, err := s.Internal("Price").Delete(ctx, ids)
	return err
}
