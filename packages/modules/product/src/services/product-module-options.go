package services

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// This file holds the product option methods of ProductModuleService
// (product-module-service.ts).

// CreateProductOptions mirrors createProductOptions.
func (s *ProductModuleService) CreateProductOptions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var err error
		out, err = t.createOptions(ctx, data)
		return err
	})
	return out, err
}

// UpdateProductOptions mirrors updateProductOptions(idOrSelector, data).
func (s *ProductModuleService) UpdateProductOptions(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		exclusive, _ := data.Get("is_exclusive")
		var input []*dal.Record
		check := func(o *dal.Record) error {
			if exclusive == true {
				if cur, _ := o.Get("is_exclusive"); cur == false {
					return invalid("Cannot change product option: %s from global to exclusive.", strOf(o, "id"))
				}
			}
			return nil
		}
		if id, ok := idOrSelector.(string); ok {
			o, err := t.Internal("ProductOption").Retrieve(ctx, id, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			input = []*dal.Record{withID(id, data)}
			if err := check(o); err != nil {
				return err
			}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			rows, err := t.Internal("ProductOption").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, o := range rows {
				if err := check(o); err != nil {
					return err
				}
				input = append(input, withID(strOf(o, "id"), data))
			}
		}
		var err error
		out, err = t.updateOptions(ctx, input)
		return err
	})
	return out, err
}

// UpsertProductOptions mirrors upsertProductOptions.
func (s *ProductModuleService) UpsertProductOptions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	// @EmitEvents runs inside the transaction upstream (before the flush):
	// the mutations wait in the shared aggregator for the next @EmitEvents
	ctx = dal.WithDeferredEmit(ctx)
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var forCreate, forUpdate []*dal.Record
		for _, o := range data {
			if strOf(o, "id") != "" {
				forUpdate = append(forUpdate, o)
			} else {
				forCreate = append(forCreate, o)
			}
		}
		if len(forCreate) > 0 {
			c, err := t.createOptions(ctx, forCreate)
			if err != nil {
				return err
			}
			out = append(out, c...)
		}
		if len(forUpdate) > 0 {
			u, err := t.updateOptions(ctx, forUpdate)
			if err != nil {
				return err
			}
			out = append(out, u...)
		}
		return nil
	})
	return out, err
}

func valueOf(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return strOf(recordOf(v), "value")
}

// updateOptions mirrors updateOptions_: values given replace the option's
// values (removed values must not be used by products), ranks are applied.
func (s *ProductModuleService) updateOptions(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	ids := make([]any, len(data))
	for i, o := range data {
		if strOf(o, "id") == "" {
			return nil, invalid("Tried to update options without specifying an ID")
		}
		ids[i] = strOf(o, "id")
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	dbOptions, err := s.Internal("ProductOption").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
	if err != nil {
		return nil, err
	}
	byID := map[string]*dal.Record{}
	for _, o := range dbOptions {
		byID[strOf(o, "id")] = o
	}
	if len(dbOptions) != len(data) {
		var missing []string
		for _, id := range ids {
			if byID[fmt.Sprint(id)] == nil {
				missing = append(missing, fmt.Sprint(id))
			}
		}
		return nil, invalid("Cannot update non-existing options with ids: %s", strings.Join(missing, ", "))
	}

	var removed []any
	for _, o := range data {
		vals, ok := o.Get("values")
		if !ok || vals == nil {
			continue
		}
		db := byID[strOf(o, "id")]
		keep := map[string]bool{}
		for _, v := range list(vals) {
			keep[valueOf(v)] = true
		}
		dbVals, _ := db.Get("values")
		for _, ev := range list(dbVals) {
			if !keep[strOf(recordOf(ev), "value")] {
				removed = append(removed, strOf(recordOf(ev), "id"))
			}
		}
	}
	if err := s.validateOptionValuesNotAssociatedWithProducts(ctx, removed); err != nil {
		return nil, err
	}

	var updated []*dal.Record
	for _, o := range data {
		db := byID[strOf(o, "id")]
		dbValsRaw, _ := db.Get("values")
		dbVals := list(dbValsRaw)
		ranks := recordOf(func() any { x, _ := o.Get("ranks"); return x }())
		vals, hasVals := o.Get("values")
		if ranks != nil {
			valid := map[string]bool{}
			if hasVals && vals != nil {
				for _, v := range list(vals) {
					valid[valueOf(v)] = true
				}
			} else {
				for _, v := range dbVals {
					valid[strOf(recordOf(v), "value")] = true
				}
			}
			for _, k := range ranks.Keys() {
				if !valid[k] {
					return nil, invalid(`Value "%s" is assigned a rank but is not defined in the list of values.`, k)
				}
			}
		}
		var normalized []any
		replace := false
		if hasVals && vals != nil {
			replace = true
			for _, v := range list(vals) {
				var vo *dal.Record
				if str, ok := v.(string); ok {
					vo = dal.NewRecord()
					vo.Set("value", str)
				} else {
					vo = clone(recordOf(v))
				}
				if ranks != nil {
					if rank, ok := ranks.Get(strOf(vo, "value")); ok {
						vo.Set("rank", rank)
					}
				}
				if strOf(vo, "id") == "" {
					for _, dv := range dbVals {
						if strOf(recordOf(dv), "value") == strOf(vo, "value") {
							vo.Set("id", strOf(recordOf(dv), "id"))
							break
						}
					}
				}
				normalized = append(normalized, vo)
			}
		} else if ranks != nil {
			replace = true
			for _, dv := range dbVals {
				d := recordOf(dv)
				vo := dal.NewRecord()
				vo.Set("id", strOf(d, "id"))
				vo.Set("value", strOf(d, "value"))
				if rank, ok := ranks.Get(strOf(d, "value")); ok {
					vo.Set("rank", rank)
				}
				normalized = append(normalized, vo)
			}
		}
		in := clone(o)
		in.Delete("ranks")
		in.Delete("values")
		if replace {
			in.Set("values", normalized)
		}
		updated = append(updated, in)
	}
	// productOptionService_.upsertWithReplace(normalizedInput, { relations: ["values"] })
	if _, err := s.Internal("ProductOption").UpsertWithReplace(ctx, updated, []string{"values"}); err != nil {
		return nil, err
	}
	f2 := dal.NewRecord()
	f2.Set("id", ids)
	return s.Internal("ProductOption").List(ctx, f2, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
}

// replaceOptionValues mirrors upsertWithReplace of the values relation
// (orphan removal deletes the values left out).
func (s *ProductModuleService) replaceOptionValues(ctx context.Context, optionID string, values []any, current []any) error {
	kept := map[string]bool{}
	var updates []any
	var creates []*dal.Record
	for _, v := range values {
		vo := recordOf(v)
		if id := strOf(vo, "id"); id != "" {
			kept[id] = true
			updates = append(updates, vo)
			continue
		}
		c := clone(vo)
		c.Set("option_id", optionID)
		creates = append(creates, c)
	}
	var removed []any
	for _, cv := range current {
		if id := strOf(recordOf(cv), "id"); !kept[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		if _, err := s.Internal("ProductOptionValue").Delete(ctx, removed); err != nil {
			return err
		}
	}
	if len(updates) > 0 {
		if _, err := s.Internal("ProductOptionValue").Update(ctx, updates); err != nil {
			return err
		}
	}
	if len(creates) > 0 {
		if _, err := s.Internal("ProductOptionValue").Create(ctx, creates); err != nil {
			return err
		}
	}
	return nil
}

// validateOptionValuesNotAssociatedWithProducts mirrors
// validateOptionValuesNotAssociatedWithProducts_.
func (s *ProductModuleService) validateOptionValuesNotAssociatedWithProducts(ctx context.Context, valueIDs []any) error {
	if len(valueIDs) == 0 {
		return nil
	}
	f := dal.NewRecord()
	f.Set("product_option_value_id", valueIDs)
	take := 1
	rows, err := s.Internal("ProductProductOptionValue").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}, Take: &take})
	if err != nil {
		return err
	}
	if len(rows) > 0 {
		return invalid("Cannot delete product option values that are associated with products.")
	}
	return nil
}

// ProductOptionProductPair mirrors ProductOptionProductPair.
type ProductOptionProductPair struct {
	ProductID       string
	ProductOptionID string
	ValueIDs        []any
	HasValueIDs     bool
}

// AddProductOptionToProduct mirrors addProductOptionToProduct.
func (s *ProductModuleService) AddProductOptionToProduct(ctx context.Context, pairs []ProductOptionProductPair) error {
	if len(pairs) == 0 {
		return nil
	}
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		in := make([]optionPair, len(pairs))
		for i, p := range pairs {
			in[i] = optionPair{productID: p.ProductID, optionID: p.ProductOptionID, valueIDs: p.ValueIDs, hasValueIDs: p.HasValueIDs}
		}
		return t.addProductOptionToProduct(ctx, in)
	})
}

// RemoveProductOptionFromProduct mirrors removeProductOptionFromProduct: it
// returns the exclusive options it orphaned (soft deleted).
func (s *ProductModuleService) RemoveProductOptionFromProduct(ctx context.Context, pairs []ProductOptionProductPair) ([]any, error) {
	var out []any
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var err error
		out, err = t.removeProductOptionFromProduct(ctx, pairs, map[string]bool{})
		return err
	})
	return out, err
}

func (s *ProductModuleService) removeProductOptionFromProduct(ctx context.Context, pairs []ProductOptionProductPair, reconciled map[string]bool) ([]any, error) {
	var or []any
	for _, p := range pairs {
		r := dal.NewRecord()
		r.Set("product_id", p.ProductID)
		r.Set("product_option_id", p.ProductOptionID)
		or = append(or, r)
	}
	f := dal.NewRecord()
	f.Set("$or", or)
	ppos, err := s.Internal("ProductProductOption").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	var cascade [][2]string
	for _, ppo := range ppos {
		pid, oid := strOf(ppo, "product_id"), strOf(ppo, "product_option_id")
		if pid != "" && oid != "" && !reconciled[pid] {
			cascade = append(cascade, [2]string{pid, oid})
		}
	}
	if len(cascade) > 0 {
		if err := s.detachOptionFromVariants(ctx, cascade); err != nil {
			return nil, err
		}
	}
	var ppoIDs []any
	var unlinked []any
	seen := map[string]bool{}
	for _, ppo := range ppos {
		ppoIDs = append(ppoIDs, strOf(ppo, "id"))
		if oid := strOf(ppo, "product_option_id"); oid != "" && !seen[oid] {
			seen[oid] = true
			unlinked = append(unlinked, oid)
		}
	}
	if len(ppoIDs) > 0 {
		var sels []any
		for _, id := range ppoIDs {
			r := dal.NewRecord()
			r.Set("product_product_option_id", id)
			sels = append(sels, r)
		}
		// productProductOptionValueService_.delete (afterDelete dispatched)
		if _, err := s.Internal("ProductProductOptionValue").Delete(ctx, sels); err != nil {
			return nil, err
		}
		if _, err := s.Internal("ProductProductOption").Delete(ctx, ppoIDs); err != nil {
			return nil, err
		}
	}
	if len(unlinked) == 0 {
		return []any{}, nil
	}
	of := dal.NewRecord()
	of.Set("id", unlinked)
	options, err := s.Internal("ProductOption").List(ctx, of, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	lf := dal.NewRecord()
	lf.Set("product_option_id", unlinked)
	remaining, err := s.Internal("ProductProductOption").List(ctx, lf, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	still := map[string]bool{}
	for _, r := range remaining {
		still[strOf(r, "product_option_id")] = true
	}
	var orphaned []any
	for _, o := range options {
		if ex, _ := o.Get("is_exclusive"); ex == true && !still[strOf(o, "id")] {
			orphaned = append(orphaned, strOf(o, "id"))
		}
	}
	if len(orphaned) > 0 {
		if _, err := s.Internal("ProductOption").SoftDelete(ctx, orphaned); err != nil {
			return nil, err
		}
	}
	if orphaned == nil {
		orphaned = []any{}
	}
	return orphaned, nil
}

// detachOptionFromVariants mirrors detachOptionFromVariants_.
func (s *ProductModuleService) detachOptionFromVariants(ctx context.Context, pairs [][2]string) error {
	var productIDs, optionIDs []any
	seenP, seenO := map[string]bool{}, map[string]bool{}
	for _, p := range pairs {
		if !seenP[p[0]] {
			seenP[p[0]] = true
			productIDs = append(productIDs, p[0])
		}
		if !seenO[p[1]] {
			seenO[p[1]] = true
			optionIDs = append(optionIDs, p[1])
		}
	}
	of := dal.NewRecord()
	of.Set("id", optionIDs)
	options, err := s.Internal("ProductOption").List(ctx, of, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
	if err != nil {
		return err
	}
	valuesByOption := map[string][]string{}
	for _, o := range options {
		vs, _ := o.Get("values")
		for _, v := range list(vs) {
			valuesByOption[strOf(o, "id")] = append(valuesByOption[strOf(o, "id")], strOf(recordOf(v), "id"))
		}
	}
	rows, err := dal.QueryRows(ctx, s.db, `select "pv"."id" as "variant_id", "pv"."title", "pv"."product_id", "pvo"."option_value_id" from "product_variant" as "pv" left join "product_variant_option" as "pvo" on "pvo"."variant_id" = "pv"."id" where "pv"."product_id" = any($1) and "pv"."deleted_at" is null`, toStrings(productIDs))
	if err != nil {
		return dal.MapDBError(err)
	}
	type variantInfo struct {
		title    any
		valueIDs map[string]bool
	}
	byProduct := map[string]map[string]*variantInfo{}
	var variantOrder = map[string][]string{}
	for _, r := range rows {
		pid, vid := fmt.Sprint(r["product_id"]), fmt.Sprint(r["variant_id"])
		if byProduct[pid] == nil {
			byProduct[pid] = map[string]*variantInfo{}
		}
		vi := byProduct[pid][vid]
		if vi == nil {
			vi = &variantInfo{title: r["title"], valueIDs: map[string]bool{}}
			byProduct[pid][vid] = vi
			variantOrder[pid] = append(variantOrder[pid], vid)
		}
		if ov := r["option_value_id"]; ov != nil {
			vi.valueIDs[fmt.Sprint(ov)] = true
		}
	}
	removedByProduct := map[string]map[string]bool{}
	var productOrder []string
	for _, p := range pairs {
		if removedByProduct[p[0]] == nil {
			removedByProduct[p[0]] = map[string]bool{}
			productOrder = append(productOrder, p[0])
		}
		for _, v := range valuesByOption[p[1]] {
			removedByProduct[p[0]][v] = true
		}
	}
	for _, pid := range productOrder {
		removed := removedByProduct[pid]
		variants := byProduct[pid]
		if len(removed) == 0 || len(variants) == 0 {
			continue
		}
		type entry struct {
			before string
			title  any
		}
		buckets := map[string][]entry{}
		var bucketOrder []string
		var toDetach []string
		for _, vid := range variantOrder[pid] {
			v := variants[vid]
			var remaining, all []string
			for id := range v.valueIDs {
				all = append(all, id)
				if !removed[id] {
					remaining = append(remaining, id)
				}
			}
			if len(remaining) != len(v.valueIDs) {
				toDetach = append(toDetach, vid)
			}
			sort.Strings(remaining)
			sort.Strings(all)
			key := strings.Join(remaining, "|")
			if _, ok := buckets[key]; !ok {
				bucketOrder = append(bucketOrder, key)
			}
			buckets[key] = append(buckets[key], entry{before: strings.Join(all, "|"), title: v.title})
		}
		for _, key := range bucketOrder {
			b := buckets[key]
			if len(b) < 2 {
				continue
			}
			distinct := map[string]bool{}
			for _, e := range b {
				distinct[e.before] = true
			}
			if len(distinct) > 1 {
				titles := make([]string, len(b))
				for i, e := range b {
					if e.title == nil {
						titles[i] = "null"
					} else {
						titles[i] = fmt.Sprint(e.title)
					}
				}
				return invalid("Cannot unassign product option from product because the following variant(s) would end up with the same option combination: %s", strings.Join(titles, ", "))
			}
		}
		if len(toDetach) > 0 {
			var removedIDs []string
			for id := range removed {
				removedIDs = append(removedIDs, id)
			}
			if _, err := dal.QueryRows(ctx, s.db, `delete from "product_variant_option" where "variant_id" = any($1) and "option_value_id" = any($2) returning "variant_id"`, toDetach, removedIDs); err != nil {
				return dal.MapDBError(err)
			}
		}
	}
	return nil
}

// ProductOptionProductValueUpdate mirrors ProductOptionProductValueUpdate:
// Add holds value ids (strings) or { value } records to create.
type ProductOptionProductValueUpdate struct {
	ProductID       string
	ProductOptionID string
	Add             []any
	Remove          []any
}

// UpdateProductOptionValuesOnProduct mirrors
// updateProductOptionValuesOnProduct.
func (s *ProductModuleService) UpdateProductOptionValuesOnProduct(ctx context.Context, updates []ProductOptionProductValueUpdate) error {
	var effective []ProductOptionProductValueUpdate
	for _, u := range updates {
		if len(u.Add) > 0 || len(u.Remove) > 0 {
			effective = append(effective, u)
		}
	}
	if len(effective) == 0 {
		return nil
	}
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var or []any
		for _, u := range effective {
			r := dal.NewRecord()
			r.Set("product_id", u.ProductID)
			r.Set("product_option_id", u.ProductOptionID)
			or = append(or, r)
		}
		f := dal.NewRecord()
		f.Set("$or", or)
		ppos, err := t.Internal("ProductProductOption").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
		if err != nil {
			return err
		}
		byKey := map[string]*dal.Record{}
		for _, p := range ppos {
			byKey[strOf(p, "product_id")+"_"+strOf(p, "product_option_id")] = p
		}
		var missing []string
		for _, u := range effective {
			if byKey[u.ProductID+"_"+u.ProductOptionID] == nil {
				missing = append(missing, u.ProductID+":"+u.ProductOptionID)
			}
		}
		if len(missing) > 0 {
			return invalid("Some product options are not linked to products: [%s]", strings.Join(missing, ", "))
		}
		for _, u := range effective {
			if len(u.Remove) == 0 {
				continue
			}
			rows, err := dal.QueryRows(ctx, t.db, `select distinct "pv"."id" as "variant_id", "pv"."title", "pvo"."option_value_id" from "product_variant" as "pv" inner join "product_variant_option" as "pvo" on "pvo"."variant_id" = "pv"."id" where "pv"."product_id" = $1 and "pvo"."option_value_id" = any($2) and "pv"."deleted_at" is null`, u.ProductID, toStrings(u.Remove))
			if err != nil {
				return dal.MapDBError(err)
			}
			var names []string
			seen := map[string]bool{}
			for _, r := range rows {
				vid := fmt.Sprint(r["variant_id"])
				if seen[vid] {
					continue
				}
				seen[vid] = true
				if r["title"] != nil && r["title"] != "" {
					names = append(names, fmt.Sprint(r["title"]))
				} else {
					names = append(names, vid)
				}
			}
			if len(names) > 0 {
				return invalid("Cannot unassign option values from product because the following variant(s) are using it: %s", strings.Join(names, ", "))
			}
		}
		normalized, err := t.normalizeProductOptionValueUpdates(ctx, effective)
		if err != nil {
			return err
		}
		var toCreate []*dal.Record
		var toDelete []any
		for _, u := range normalized {
			ppo := byKey[u.ProductID+"_"+u.ProductOptionID]
			existing := map[string]bool{}
			vs, _ := ppo.Get("values")
			for _, v := range list(vs) {
				existing[strOf(recordOf(v), "id")] = true
			}
			added := map[string]bool{}
			for _, id := range u.Add {
				k := fmt.Sprint(id)
				if existing[k] || added[k] {
					continue
				}
				added[k] = true
				r := dal.NewRecord()
				r.Set("product_product_option_id", strOf(ppo, "id"))
				r.Set("product_option_value_id", k)
				toCreate = append(toCreate, r)
			}
			removed := map[string]bool{}
			for _, id := range u.Remove {
				k := fmt.Sprint(id)
				if removed[k] {
					continue
				}
				removed[k] = true
				sel := dal.NewRecord()
				sel.Set("product_product_option_id", strOf(ppo, "id"))
				sel.Set("product_option_value_id", k)
				toDelete = append(toDelete, sel)
			}
		}
		if len(toDelete) > 0 {
			if _, err := t.Internal("ProductProductOptionValue").Delete(ctx, toDelete); err != nil {
				return err
			}
		}
		if len(toCreate) > 0 {
			if _, err := t.Internal("ProductProductOptionValue").Create(ctx, toCreate); err != nil {
				return err
			}
		}
		return nil
	})
}

// normalizeProductOptionValueUpdates mirrors
// normalizeProductOptionValueUpdates_: { value } entries are created on
// the option (when missing) and replaced by their ids.
func (s *ProductModuleService) normalizeProductOptionValueUpdates(ctx context.Context, updates []ProductOptionProductValueUpdate) ([]ProductOptionProductValueUpdate, error) {
	namesByOption := map[string][]string{}
	var optionOrder []string
	existingIDs := make([][]any, len(updates))
	names := make([][]string, len(updates))
	for i, u := range updates {
		for _, a := range u.Add {
			if str, ok := a.(string); ok {
				existingIDs[i] = append(existingIDs[i], str)
				continue
			}
			v := strings.TrimSpace(strOf(recordOf(a), "value"))
			if v == "" {
				continue
			}
			if _, ok := namesByOption[u.ProductOptionID]; !ok {
				optionOrder = append(optionOrder, u.ProductOptionID)
			}
			namesByOption[u.ProductOptionID] = append(namesByOption[u.ProductOptionID], v)
			names[i] = append(names[i], v)
		}
	}
	out := make([]ProductOptionProductValueUpdate, len(updates))
	if len(namesByOption) == 0 {
		for i, u := range updates {
			out[i] = u
			out[i].Add = existingIDs[i]
		}
		return out, nil
	}
	ids := make([]any, len(optionOrder))
	for i, id := range optionOrder {
		ids[i] = id
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	options, err := s.Internal("ProductOption").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"values"}})
	if err != nil {
		return nil, err
	}
	byID := map[string]*dal.Record{}
	for _, o := range options {
		byID[strOf(o, "id")] = o
	}
	for _, id := range optionOrder {
		if byID[id] == nil {
			return nil, invalid("Cannot find product option with id: %s", id)
		}
	}
	var toUpdate []*dal.Record
	for _, id := range optionOrder {
		o := byID[id]
		vs, _ := o.Get("values")
		existing := map[string]bool{}
		var all []any
		for _, v := range list(vs) {
			existing[strOf(recordOf(v), "value")] = true
			all = append(all, strOf(recordOf(v), "value"))
		}
		added := false
		seen := map[string]bool{}
		for _, n := range namesByOption[id] {
			if !existing[n] && !seen[n] {
				seen[n] = true
				all = append(all, n)
				added = true
			}
		}
		if added {
			u := dal.NewRecord()
			u.Set("id", id)
			u.Set("values", all)
			toUpdate = append(toUpdate, u)
		}
	}
	if len(toUpdate) > 0 {
		updated, err := s.updateOptions(ctx, toUpdate)
		if err != nil {
			return nil, err
		}
		for _, o := range updated {
			byID[strOf(o, "id")] = o
		}
	}
	for i, u := range updates {
		out[i] = u
		add := append([]any{}, existingIDs[i]...)
		o := byID[u.ProductOptionID]
		if o != nil {
			vs, _ := o.Get("values")
			for _, n := range names[i] {
				for _, v := range list(vs) {
					if strOf(recordOf(v), "value") == n {
						add = append(add, strOf(recordOf(v), "id"))
					}
				}
			}
		}
		out[i].Add = add
	}
	return out, nil
}

var _ = common.NewMedusaError

// UpdateProductOptionValues mirrors updateProductOptionValues(id, data).
func (s *ProductModuleService) UpdateProductOptionValues(ctx context.Context, id string, data *dal.Record) (*dal.Record, error) {
	if _, err := s.Internal("ProductOptionValue").Retrieve(ctx, id, utilsmodulessdk.FindConfig{}); err != nil {
		return nil, err
	}
	res, err := s.Internal("ProductOptionValue").Update(ctx, []any{withID(id, data)})
	if err != nil {
		return nil, err
	}
	rows := records(res)
	if len(rows) == 0 {
		return nil, nil
	}
	return rows[0], nil
}

// SoftDeleteProductOptionValues mirrors softDeleteProductOptionValues.
func (s *ProductModuleService) SoftDeleteProductOptionValues(ctx context.Context, ids []any) error {
	// no @EmitEvents upstream: the mutations emit no events
	ctx = dal.WithoutEmit(ctx)
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		if err := t.validateOptionValuesNotAssociatedWithProducts(ctx, ids); err != nil {
			return err
		}
		_, err := t.Internal("ProductOptionValue").SoftDelete(ctx, ids)
		return err
	})
}
