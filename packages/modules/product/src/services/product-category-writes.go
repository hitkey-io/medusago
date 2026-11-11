package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// This file mirrors the write side of ProductCategoryService and
// ProductCategoryRepository (repositories/product-category.ts): the tree
// (mpath) and the sibling ranks.

func numberOf(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	}
	return 0, false
}

func (s *ProductModuleService) categories() *utilsmodulessdk.InternalService {
	return s.Internal("ProductCategory")
}

// countSiblings mirrors manager.count(ProductCategory, { parent_category_id }).
func (s *ProductModuleService) countSiblings(ctx context.Context, parentID any) (int, error) {
	f := dal.NewRecord()
	f.Set("parent_category_id", parentID)
	_, n, err := s.categories().ListAndCount(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
	return n, err
}

func (s *ProductModuleService) findCategory(ctx context.Context, id any) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	rows, err := s.categories().List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return rows[0], nil
}

// shiftRanks adds delta to the rank of the siblings matching the rank
// condition (rerankSiblingsAfterCreation / AfterDeletion / rerankAllSiblings).
type undefinedParent struct{}

// anyParent stands for an undefined parent_category_id: MikroORM drops
// the key from the where clause, so every category matches (upstream
// rerankSiblingsAfterCreation with a category created without parent).
var anyParent = undefinedParent{}

func (s *ProductModuleService) shiftRanks(ctx context.Context, parentID any, rankCond *dal.Record, delta float64) error {
	f := dal.NewRecord()
	if _, any := parentID.(undefinedParent); !any {
		f.Set("parent_category_id", parentID)
	}
	f.Set("rank", rankCond)
	rows, err := s.categories().List(ctx, f, utilsmodulessdk.FindConfig{Order: orderRankAsc()})
	if err != nil {
		return err
	}
	var updates []any
	for _, r := range rows {
		rank, _ := numberOf(func() any { x, _ := r.Get("rank"); return x }())
		u := dal.NewRecord()
		u.Set("id", strOf(r, "id"))
		u.Set("rank", rank+delta)
		updates = append(updates, u)
	}
	if len(updates) == 0 {
		return nil
	}
	_, err = s.categories().Update(ctx, updates)
	return err
}

func orderRankAsc() []dal.OrderBy {
	return []dal.OrderBy{{Field: "rank", Direction: "ASC"}}
}

func cond(op string, v any) *dal.Record {
	r := dal.NewRecord()
	r.Set(op, v)
	return r
}

// CreateProductCategories mirrors createProductCategories.
func (s *ProductModuleService) CreateProductCategories(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(dal.WithoutEvents(ctx), func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var err error
		out, err = t.createCategories(ctx, data)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.categoryEvent(ctx, "created", out)
	return out, nil
}

// categoryEvent mirrors eventBuilders.<action>ProductCategory: the custom
// category service writes outside the entity subscriber, so the category
// events are the explicitly built ones.
func (s *ProductModuleService) categoryEvent(ctx context.Context, action string, categories []*dal.Record) {
	ids := make([]any, len(categories))
	for i, c := range categories {
		ids[i], _ = c.Get("id")
	}
	s.SaveEvent(ctx, utilsmodulessdk.ModuleEventBuilderFactory(utilsmodulessdk.PRODUCT, action, "product_category", "product.product-category."+action)(ids...))
}

func (s *ProductModuleService) createCategories(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	for i, entry := range data {
		c := clone(entry)
		if v, _ := c.Get("handle"); v == nil {
			c.Set("handle", KebabCase(strOf(c, "name")))
		}
		parentID, _ := c.Get("parent_category_id")
		if parentID == "" {
			parentID = nil
		}
		siblings, err := s.countSiblings(ctx, parentID)
		if err != nil {
			return nil, err
		}
		max := float64(siblings + i)
		rank, hasRank := numberOf(func() any { x, _ := c.Get("rank"); return x }())
		if !hasRank || rank > max {
			rank = max
		}
		c.Set("rank", rank)
		if rank < max {
			rerankParent := any(anyParent)
			if raw, given := entry.Get("parent_category_id"); given {
				rerankParent = raw
			}
			if err := s.shiftRanks(ctx, rerankParent, cond("$gte", rank), 1); err != nil {
				return nil, err
			}
		}
		var parent *dal.Record
		if parentID != nil {
			parent, err = s.findCategory(ctx, parentID)
			if err != nil {
				return nil, err
			}
			if parent == nil {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, fmt.Sprintf("Parent category with id: '%v' does not exist", parentID))
			}
		}
		id := common.GenerateEntityID(strOf(c, "id"), "pcat")
		c.Set("id", id)
		if parent != nil {
			c.Set("mpath", strOf(parent, "mpath")+"."+id)
		} else {
			c.Set("mpath", id)
		}
		created, err := s.categories().Create(ctx, []*dal.Record{c})
		if err != nil {
			return nil, err
		}
		out = append(out, created...)
	}
	return out, nil
}

// UpdateProductCategories mirrors updateProductCategories(idOrSelector, data).
func (s *ProductModuleService) UpdateProductCategories(ctx context.Context, idOrSelector any, data *dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	defer func() {
		if out != nil {
			s.categoryEvent(ctx, "updated", out)
		}
	}()
	err := s.InTransaction(dal.WithoutEvents(ctx), func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var input []*dal.Record
		if id, ok := idOrSelector.(string); ok {
			if _, err := t.categories().Retrieve(ctx, id, utilsmodulessdk.FindConfig{}); err != nil {
				return err
			}
			input = []*dal.Record{withID(id, data)}
		} else {
			sel, _ := idOrSelector.(*dal.Record)
			rows, err := t.categories().List(ctx, sel, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			for _, r := range rows {
				input = append(input, withID(strOf(r, "id"), data))
			}
		}
		var err error
		out, err = t.updateCategories(ctx, input)
		return err
	})
	return out, err
}

// UpsertProductCategories mirrors upsertProductCategories.
func (s *ProductModuleService) UpsertProductCategories(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out, created, updated []*dal.Record
	defer func() {
		if out != nil {
			s.categoryEvent(ctx, "created", created)
			s.categoryEvent(ctx, "updated", updated)
		}
	}()
	err := s.InTransaction(dal.WithoutEvents(ctx), func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		var forCreate, forUpdate []*dal.Record
		for _, c := range data {
			if strOf(c, "id") != "" {
				forUpdate = append(forUpdate, c)
			} else {
				forCreate = append(forCreate, c)
			}
		}
		if len(forCreate) > 0 {
			c, err := t.createCategories(ctx, forCreate)
			if err != nil {
				return err
			}
			created = c
			out = append(out, c...)
		}
		if len(forUpdate) > 0 {
			u, err := t.updateCategories(ctx, forUpdate)
			if err != nil {
				return err
			}
			updated = u
			out = append(out, u...)
		}
		return nil
	})
	return out, err
}

// definedOnly drops the keys whose value is undefined (nil keeps null
// semantics only for explicitly given keys, like isDefined).
func (s *ProductModuleService) updateCategories(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	for i, entry := range data {
		c := clone(entry)
		id := strOf(c, "id")
		current, err := s.findCategory(ctx, id)
		if err != nil {
			return nil, err
		}
		if current == nil {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("ProductCategory with id: %s was not found", id))
		}
		newParent, parentGiven := c.Get("parent_category_id")
		rank, rankGiven := numberOf(func() any { x, _ := c.Get("rank"); return x }())
		// isDefined(null) is true: a null parent moves the category to the root.
		curParent, _ := current.Get("parent_category_id")
		curRank, _ := numberOf(func() any { x, _ := current.Get("rank"); return x }())
		var mpathUpdates []any
		switch {
		case parentGiven && fmt.Sprint(newParent) != fmt.Sprint(curParent):
			if newParent == nil {
				c.Set("mpath", "")
			} else {
				parent, err := s.findCategory(ctx, newParent)
				if err != nil {
					return nil, err
				}
				if parent == nil {
					return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, fmt.Sprintf("Parent category with id: '%v' does not exist", newParent))
				}
				base := strOf(parent, "mpath") + "." + id
				c.Set("mpath", base)
				oldMpath := strOf(current, "mpath")
				desc, err := dal.QueryRows(ctx, s.db, `select "id", "mpath" from "product_category" where "mpath" like $1 and "deleted_at" is null`, oldMpath+".%")
				if err != nil {
					return nil, dal.MapDBError(err)
				}
				for _, d := range desc {
					u := dal.NewRecord()
					u.Set("id", d["id"])
					u.Set("mpath", base+fmt.Sprint(d["mpath"])[len(oldMpath):])
					mpathUpdates = append(mpathUpdates, u)
				}
			}
			siblings, err := s.countSiblings(ctx, newParent)
			if err != nil {
				return nil, err
			}
			max := float64(siblings + i)
			if !rankGiven || rank > max {
				rank = max
			}
			c.Set("rank", rank)
			if rank < max {
				if err := s.shiftRanks(ctx, newParent, cond("$gte", rank), 1); err != nil {
					return nil, err
				}
			}
			if err := s.shiftRanks(ctx, curParent, cond("$gt", curRank), -1); err != nil {
				return nil, err
			}
		case rankGiven:
			siblings, err := s.countSiblings(ctx, curParent)
			if err != nil {
				return nil, err
			}
			max := float64(siblings - 1 + i)
			if rank > max {
				rank = max
			}
			c.Set("rank", rank)
			if rank != curRank {
				r := dal.NewRecord()
				if curRank < rank {
					r.Set("$gt", curRank)
					r.Set("$lte", rank)
					if err := s.shiftRanks(ctx, curParent, r, -1); err != nil {
						return nil, err
					}
				} else {
					r.Set("$gte", rank)
					r.Set("$lt", curRank)
					if err := s.shiftRanks(ctx, curParent, r, 1); err != nil {
						return nil, err
					}
				}
			}
		}
		if _, err := s.categories().Update(ctx, []any{c}); err != nil {
			return nil, err
		}
		if len(mpathUpdates) > 0 {
			if _, err := s.categories().Update(ctx, mpathUpdates); err != nil {
				return nil, err
			}
		}
		updated, err := s.findCategory(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, updated)
	}
	return out, nil
}

// SoftDeleteProductCategories mirrors softDeleteProductCategories (the
// category repository softDelete: no children allowed, siblings reranked).
func (s *ProductModuleService) SoftDeleteProductCategories(ctx context.Context, ids []any) error {
	err := s.InTransaction(dal.WithoutEvents(ctx), func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		if err := t.baseDeleteCategories(ctx, ids); err != nil {
			return err
		}
		_, err := t.categories().SoftDelete(ctx, ids)
		return err
	})
	if err != nil {
		return err
	}
	deleted := make([]*dal.Record, len(ids))
	for i, id := range ids {
		deleted[i] = dal.NewRecord()
		deleted[i].Set("id", id)
	}
	s.categoryEvent(ctx, "deleted", deleted)
	return nil
}

// DeleteProductCategories mirrors deleteProductCategories (hard delete).
func (s *ProductModuleService) DeleteProductCategories(ctx context.Context, ids []any) error {
	return s.InTransaction(dal.WithoutEvents(ctx), func(tx *utilsmodulessdk.MedusaService) error {
		t := &ProductModuleService{MedusaService: tx, db: tx.DB()}
		if err := t.baseDeleteCategories(ctx, ids); err != nil {
			return err
		}
		_, err := t.categories().Delete(ctx, ids)
		return err
	})
}

// RestoreProductCategories mirrors restoreProductCategories.
func (s *ProductModuleService) RestoreProductCategories(ctx context.Context, ids []any) error {
	_, err := s.categories().Restore(dal.WithoutEvents(ctx), ids)
	return err
}

func (s *ProductModuleService) baseDeleteCategories(ctx context.Context, ids []any) error {
	for _, id := range ids {
		c, err := s.findCategory(ctx, id)
		if err != nil {
			return err
		}
		if c == nil {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("ProductCategory with id: %v was not found", id))
		}
		f := dal.NewRecord()
		f.Set("parent_category_id", id)
		_, n, err := s.categories().ListAndCount(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		if n > 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Deleting ProductCategory (%v) with category children is not allowed", id))
		}
		parent, _ := c.Get("parent_category_id")
		rank, _ := numberOf(func() any { x, _ := c.Get("rank"); return x }())
		if err := s.shiftRanks(ctx, parent, cond("$gt", rank), -1); err != nil {
			return err
		}
	}
	return nil
}
