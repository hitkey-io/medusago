package steps

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// LinkWorkflowInput mirrors LinkWorkflowInput / BatchUpdateProductsOnCategoryWorkflowInput.
type LinkWorkflowInput struct {
	ID     string
	Add    []any
	Remove []any
}

type categoryLinkComp struct {
	id         string
	add        []any
	remove     []any
	productIDs []any
}

func categoryIDsOf(p *dal.Record) []any {
	cats, _ := p.Get("categories")
	var out []any
	for _, c := range anyList(cats) {
		if r, ok := c.(*dal.Record); ok {
			id, _ := r.Get("id")
			out = append(out, id)
		}
	}
	return out
}

func setCategories(ctx composer.StepExecutionContext, productIDs []any, categoryID string, removeSet map[string]bool, invert bool) error {
	svc := ProductService(ctx.Container)
	f := dal.NewRecord()
	f.Set("id", productIDs)
	products, err := svc.ListBySuffix(ctx.Context, "Products", f, utilsmodulessdk.FindConfig{Select: []string{"id"}, Relations: []string{"categories"}})
	if err != nil {
		return err
	}
	var upserts []*dal.Record
	for _, p := range products {
		pid, _ := p.Get("id")
		current := categoryIDsOf(p)
		removing := removeSet[fmt.Sprint(pid)]
		if invert {
			removing = !removing
		}
		ids := []any{}
		if removing {
			for _, c := range current {
				if fmt.Sprint(c) != categoryID {
					ids = append(ids, c)
				}
			}
		} else {
			ids = append(append(ids, current...), categoryID)
		}
		u := dal.NewRecord()
		u.Set("id", pid)
		u.Set("category_ids", ids)
		upserts = append(upserts, u)
	}
	if len(upserts) == 0 {
		return nil
	}
	_, err = svc.UpsertProducts(ctx.Context, upserts)
	return err
}

// BatchLinkProductsToCategoryStep mirrors batchLinkProductsToCategoryStep.
var BatchLinkProductsToCategoryStep = composer.CreateStep("batch-link-products-to-category",
	func(ctx composer.StepExecutionContext, in LinkWorkflowInput) (composer.StepResponse[any], error) {
		if len(in.Add) == 0 && len(in.Remove) == 0 {
			return composer.NewStepResponse[any](nil, any(nil)), nil
		}
		removeSet := map[string]bool{}
		for _, id := range in.Remove {
			removeSet[fmt.Sprint(id)] = true
		}
		ids := append(append([]any{}, in.Add...), in.Remove...)
		if err := setCategories(ctx, ids, in.ID, removeSet, false); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(categoryLinkComp{id: in.ID, add: in.Add, remove: in.Remove, productIDs: ids})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(categoryLinkComp)
		if !ok {
			return nil
		}
		removeSet := map[string]bool{}
		for _, id := range c.remove {
			removeSet[fmt.Sprint(id)] = true
		}
		return setCategories(ctx, c.productIDs, c.id, removeSet, true)
	})

type collectionLinkComp struct {
	id         string
	productIDs []any
}

// BatchLinkProductsToCollectionStep mirrors batchLinkProductsToCollectionStep.
var BatchLinkProductsToCollectionStep = composer.CreateStep("batch-link-products-to-collection",
	func(ctx composer.StepExecutionContext, in LinkWorkflowInput) (composer.StepResponse[any], error) {
		if len(in.Add) == 0 && len(in.Remove) == 0 {
			return composer.NewStepResponse[any](nil, any(nil)), nil
		}
		svc := ProductService(ctx.Container)
		col, err := svc.Internal("ProductCollection").Retrieve(ctx.Context, in.ID, utilsmodulessdk.FindConfig{Select: []string{"id", "products.id"}, Relations: []string{"products"}})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		ps, _ := col.Get("products")
		existing := []any{}
		for _, p := range anyList(ps) {
			if r, ok := p.(*dal.Record); ok {
				id, _ := r.Get("id")
				existing = append(existing, id)
			}
		}
		remove := map[string]bool{}
		for _, id := range in.Remove {
			remove[fmt.Sprint(id)] = true
		}
		newIDs := []any{}
		for _, id := range existing {
			if !remove[fmt.Sprint(id)] {
				newIDs = append(newIDs, id)
			}
		}
		newIDs = append(newIDs, in.Add...)
		u := dal.NewRecord()
		u.Set("product_ids", newIDs)
		if _, err := svc.UpdateProductCollections(ctx.Context, in.ID, u); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(collectionLinkComp{id: in.ID, productIDs: existing})), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		c, ok := data.(collectionLinkComp)
		if !ok {
			return nil
		}
		u := dal.NewRecord()
		u.Set("product_ids", c.productIDs)
		_, err := ProductService(ctx.Container).UpdateProductCollections(ctx.Context, c.id, u)
		return err
	})
