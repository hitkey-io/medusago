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
