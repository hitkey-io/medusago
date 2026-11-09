package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// UpdateProducts mirrors updateProducts(idOrSelector, data).
func (s *ProductModuleService) UpdateProducts(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		input, err := t.normalizeSelectorUpdate(ctx, "Product", idOrSelector, data)
		if err != nil {
			return err
		}
		products, err := t.updateProducts(ctx, input)
		if err != nil {
			return err
		}
		if err := t.filterOptionValues(ctx, products); err != nil {
			return err
		}
		out = products
		return nil
	})
	return out, err
}

// UpsertProducts mirrors upsertProducts.
func (s *ProductModuleService) UpsertProducts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var forCreate, forUpdate []*dal.Record
	for _, p := range data {
		if id, _ := p.Get("id"); id != nil && id != "" {
			forUpdate = append(forUpdate, p)
		} else {
			forCreate = append(forCreate, p)
		}
	}
	var created, updated []*dal.Record
	var err error
	if len(forCreate) > 0 {
		if created, err = s.CreateProducts(ctx, forCreate); err != nil {
			return nil, err
		}
	}
	if len(forUpdate) > 0 {
		err = s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
			t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
			var uerr error
			updated, uerr = t.updateProducts(ctx, forUpdate)
			return uerr
		})
		if err != nil {
			return nil, err
		}
	}
	return append(created, updated...), nil
}

// productScalarKeys are the update keys assigned to the product row itself.
var productRelationKeys = map[string]bool{
	"options": true, "variants": true, "tags": true, "categories": true, "images": true,
	"option_ids": true, "type": true, "collection": true,
}

// updateProducts mirrors updateProducts_ and ProductRepository.deepUpdate
// for scalar fields, tags, categories and images. Option links
// (option_ids) and nested variants are not supported yet.
func (s *ProductModuleService) updateProducts(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	for _, p := range data {
		if p.Has("option_ids") || p.Has("variants") {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Updating product options or variants through the product is not supported yet")
		}
	}
	normalized := make([]*dal.Record, len(data))
	for i, p := range data {
		normalized[i] = normalizeUpdateProductInput(p)
		if h := strOf(normalized[i], "handle"); h != "" && !IsValidHandle(h) {
			return nil, invalid("Invalid product handle '%s'. It must contain URL safe characters", h)
		}
	}

	idList := make([]string, len(normalized))
	for i, p := range normalized {
		idList[i] = strOf(p, "id")
	}
	relations := map[string]bool{}
	for _, p := range normalized {
		for _, k := range []string{"tags", "categories", "images", "collection", "type"} {
			if p.Has(k) {
				relations[k] = true
			}
		}
	}
	var relList []string
	for _, k := range []string{"tags", "categories", "images", "collection", "type"} {
		if relations[k] {
			relList = append(relList, k)
		}
	}
	existing, err := s.findByIDsWithSplitPopulate(ctx, idList, relList)
	if err != nil {
		return nil, err
	}
	byID := map[string]*dal.Record{}
	for _, p := range existing {
		byID[strOf(p, "id")] = p
	}
	var missing []string
	for _, id := range idList {
		if byID[id] == nil {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Unable to update the products with ids: %s", strings.Join(missing, ", ")))
	}

	var scalarUpdates []any
	for _, p := range normalized {
		id, _ := p.Get("id")
		row := dal.NewRecord()
		row.Set("id", id)
		for _, k := range p.Keys() {
			if productRelationKeys[k] {
				continue
			}
			v, _ := p.Get(k)
			row.Set(k, v)
		}
		scalarUpdates = append(scalarUpdates, row)

		if tags, ok := p.Get("tags"); ok {
			if err := s.Internal("Product").ReplaceManyToMany(ctx, id, "tags", productIDsOf(tags)); err != nil {
				return nil, err
			}
		}
		if cats, ok := p.Get("categories"); ok {
			if err := s.Internal("Product").ReplaceManyToMany(ctx, id, "categories", productIDsOf(cats)); err != nil {
				return nil, err
			}
		}
		if images, ok := p.Get("images"); ok {
			if err := s.replaceImages(ctx, id, records(images), byID[fmt.Sprint(id)]); err != nil {
				return nil, err
			}
		}
	}
	if _, err := s.Internal("Product").Update(ctx, scalarUpdates); err != nil {
		return nil, err
	}
	return s.findByIDsWithSplitPopulate(ctx, idList, append([]string{"options", "options.values", "variants", "images", "tags"}, relList...))
}

// replaceImages mirrors assigning images (with rank = index) to the
// product: existing images are updated, new ones created and the removed
// ones deleted (orphanRemoval).
func (s *ProductModuleService) replaceImages(ctx context.Context, productID any, images []*dal.Record, product *dal.Record) error {
	current := map[string]bool{}
	if product != nil {
		cur, _ := product.Get("images")
		for _, img := range records(cur) {
			current[strOf(img, "id")] = true
		}
	}
	kept := map[string]bool{}
	var updates []any
	var creates []*dal.Record
	for i, img := range images {
		id := strOf(img, "id")
		if id != "" && current[id] {
			kept[id] = true
			u := clone(img)
			u.Set("rank", float64(i))
			updates = append(updates, u)
			continue
		}
		c := clone(img)
		c.Delete("id")
		c.Set("rank", float64(i))
		c.Set("product_id", productID)
		creates = append(creates, c)
	}
	var removed []any
	for id := range current {
		if !kept[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		if _, err := s.Internal("ProductImage").Delete(ctx, removed); err != nil {
			return err
		}
	}
	if len(updates) > 0 {
		if _, err := s.Internal("ProductImage").Update(ctx, updates); err != nil {
			return err
		}
	}
	if len(creates) > 0 {
		if _, err := s.Internal("ProductImage").Create(ctx, creates); err != nil {
			return err
		}
	}
	return nil
}

// UpdateProductVariants mirrors updateProductVariants(idOrSelector, data).
func (s *ProductModuleService) UpdateProductVariants(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var input []*dal.Record
		if id, ok := idOrSelector.(string); ok {
			input = []*dal.Record{withID(id, data)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			rows, err := t.Internal("ProductVariant").List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, r := range rows {
				id, _ := r.Get("id")
				input = append(input, withID(id, data))
			}
		}
		var err error
		out, err = t.updateVariants(ctx, input)
		return err
	})
	return out, err
}

// UpsertProductVariants mirrors upsertProductVariants.
func (s *ProductModuleService) UpsertProductVariants(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var forCreate, forUpdate []*dal.Record
	for _, v := range data {
		if id, _ := v.Get("id"); id != nil && id != "" {
			forUpdate = append(forUpdate, v)
		} else {
			forCreate = append(forCreate, v)
		}
	}
	var out []*dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		if len(forCreate) > 0 {
			created, err := t.createVariants(ctx, forCreate)
			if err != nil {
				return err
			}
			out = append(out, created...)
		}
		if len(forUpdate) > 0 {
			updated, err := t.updateVariants(ctx, forUpdate)
			if err != nil {
				return err
			}
			out = append(out, updated...)
		}
		return nil
	})
	return out, err
}

// updateVariants mirrors updateVariants_.
func (s *ProductModuleService) updateVariants(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	if len(data) == 0 {
		return []*dal.Record{}, nil
	}
	hasOptions := false
	ids := make([]any, len(data))
	for i, d := range data {
		ids[i], _ = d.Get("id")
		if o, ok := d.Get("options"); ok && isPresentValue(o) {
			hasOptions = true
		}
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	variants, err := s.Internal("ProductVariant").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(variants) != len(data) {
		found := map[string]bool{}
		for _, v := range variants {
			found[strOf(v, "id")] = true
		}
		var missing []string
		for _, id := range ids {
			if !found[fmt.Sprint(id)] {
				missing = append(missing, fmt.Sprint(id))
			}
		}
		return nil, invalid("Cannot update non-existing variants with ids: %s", strings.Join(missing, ", "))
	}
	withProduct := make([]*dal.Record, 0, len(variants))
	for _, v := range variants {
		var update *dal.Record
		for _, d := range data {
			if fmt.Sprint(func() any { x, _ := d.Get("id"); return x }()) == strOf(v, "id") {
				update = d
				break
			}
		}
		r := dal.NewRecord()
		for _, k := range update.Keys() {
			if k == "options" || k == "id" {
				continue
			}
			x, _ := update.Get(k)
			r.Set(k, x)
		}
		if o, ok := update.Get("options"); ok && isPresentValue(o) {
			r.Set("options", o)
		}
		r.Set("id", strOf(v, "id"))
		pid, _ := v.Get("product_id")
		r.Set("product_id", pid)
		withProduct = append(withProduct, r)
	}
	if hasOptions {
		var pids []any
		seen := map[string]bool{}
		for _, v := range variants {
			pid := strOf(v, "product_id")
			if !seen[pid] {
				seen[pid] = true
				pids = append(pids, pid)
			}
		}
		pf := dal.NewRecord()
		pf.Set("product_id", pids)
		all, err := s.Internal("ProductVariant").List(ctx, pf, utilsmodulessdk.FindConfig{Relations: []string{"options"}})
		if err != nil {
			return nil, err
		}
		optionsByProduct, valueIDsByProduct, err := s.loadOptionsAndValuesByProductID(ctx, pids)
		if err != nil {
			return nil, err
		}
		if withProduct, err = s.assignOptionsToVariants(withProduct, all, optionsByProduct, valueIDsByProduct); err != nil {
			return nil, err
		}
	}
	var updates []any
	for _, v := range withProduct {
		u := clone(v)
		u.Delete("options")
		updates = append(updates, u)
	}
	res, err := s.Internal("ProductVariant").Update(ctx, updates)
	if err != nil {
		return nil, err
	}
	if hasOptions {
		for _, v := range withProduct {
			o, ok := v.Get("options")
			if !ok {
				continue
			}
			if err := s.Internal("ProductVariant").ReplaceManyToMany(ctx, strOf(v, "id"), "options", productIDsOf(o)); err != nil {
				return nil, err
			}
		}
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	return records(res), nil
}

// isPresentValue mirrors isPresent: not null/undefined and not an empty
// object, array or string.
func isPresentValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t != nil && t.Len() > 0
	}
	return true
}
