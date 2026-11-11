// Package workflows mirrors @medusajs/core-flows/src/product-category/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product-category/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		ev := dal.NewRecord()
		id, _ := r.Get("id")
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

// CreateProductCategoriesInput mirrors CreateProductCategoriesWorkflowInput.
type CreateProductCategoriesInput struct {
	ProductCategories []*dal.Record
	AdditionalData    *dal.Record
}

// CreateProductCategoriesWorkflow mirrors createProductCategoriesWorkflow.
var CreateProductCategoriesWorkflow = composer.CreateWorkflow("create-product-categories",
	func(e *composer.Execution, input CreateProductCategoriesInput) ([]*dal.Record, error) {
		created, err := steps.CreateProductCategoriesStep.Run(e, input.ProductCategories)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-category.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("categories", created)
		hook.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "categoriesCreated", hook); err != nil {
			return nil, err
		}
		return created, nil
	})

// UpdateProductCategoriesInput mirrors UpdateProductCategoriesWorkflowInput.
type UpdateProductCategoriesInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	AdditionalData *dal.Record
}

// UpdateProductCategoriesWorkflow mirrors updateProductCategoriesWorkflow.
var UpdateProductCategoriesWorkflow = composer.CreateWorkflow("update-product-categories",
	func(e *composer.Execution, input UpdateProductCategoriesInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateProductCategoriesStep.Run(e, steps.UpdateProductCategoriesStepInput{Selector: input.Selector, Update: input.Update})
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-category.updated", Data: idEvents(updated)}); err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("categories", updated)
		hook.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "categoriesUpdated", hook); err != nil {
			return nil, err
		}
		return updated, nil
	})

// DeleteProductCategoriesWorkflow mirrors deleteProductCategoriesWorkflow
// (input: the ids).
var DeleteProductCategoriesWorkflow = composer.CreateWorkflow("delete-product-categories",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteProductCategoriesStep.Run(e, ids); err != nil {
			return nil, err
		}
		events := make([]any, len(ids))
		for i, id := range ids {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		err := composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-category.deleted", Data: events})
				return err
			},
			func() error {
				inner := dal.NewRecord()
				inner.Set("product_category_id", ids)
				entry := dal.NewRecord()
				entry.Set(utilsmodulessdk.PRODUCT, inner)
				_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("ids", ids)
		if err := composer.CreateHook(e, "categoriesDeleted", hook); err != nil {
			return nil, err
		}
		return nil, nil
	})
