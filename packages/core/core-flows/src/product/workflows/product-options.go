package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// CreateProductOptionsInput mirrors CreateProductOptionsWorkflowInput.
type CreateProductOptionsInput struct {
	ProductOptions []*dal.Record
	AdditionalData *dal.Record
}

// CreateProductOptionsWorkflow mirrors createProductOptionsWorkflow.
var CreateProductOptionsWorkflow = composer.CreateWorkflow("create-product-options",
	func(e *composer.Execution, input CreateProductOptionsInput) ([]*dal.Record, error) {
		created, err := steps.CreateProductOptionsStep.Run(e, input.ProductOptions)
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_options", created)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productOptionsCreated", hookInput); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-option.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		return created, nil
	})

// UpdateProductOptionsInput mirrors UpdateProductOptionsWorkflowInput.
type UpdateProductOptionsInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	AdditionalData *dal.Record
}

// UpdateProductOptionsWorkflow mirrors updateProductOptionsWorkflow.
var UpdateProductOptionsWorkflow = composer.CreateWorkflow("update-product-options",
	func(e *composer.Execution, input UpdateProductOptionsInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateProductOptionsStep.Run(e, steps.UpdateBySelectorInput{Selector: input.Selector, Update: input.Update})
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_options", updated)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productOptionsUpdated", hookInput); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-option.updated", Data: idEvents(updated)}); err != nil {
			return nil, err
		}
		return updated, nil
	})

// SetProductProductOptionsInput mirrors SetProductProductOptionsWorkflowInput.
// Add holds option ids (strings) or {product_option_id,
// product_option_value_ids} entries.
type SetProductProductOptionsInput struct {
	ProductID string
	Add       []services.ProductOptionProductPair
	Remove    []string
	Update    []services.ProductOptionProductValueUpdate
}

// SetProductProductOptionsWorkflow mirrors setProductProductOptionsWorkflow.
var SetProductProductOptionsWorkflow = composer.CreateWorkflow("set-product-product-options",
	func(e *composer.Execution, input SetProductProductOptionsInput) (any, error) {
		toAdd := make([]services.ProductOptionProductPair, len(input.Add))
		for i, a := range input.Add {
			a.ProductID = input.ProductID
			toAdd[i] = a
		}
		toRemove := make([]services.ProductOptionProductPair, len(input.Remove))
		for i, id := range input.Remove {
			toRemove[i] = services.ProductOptionProductPair{ProductID: input.ProductID, ProductOptionID: id}
		}
		toUpdate := make([]services.ProductOptionProductValueUpdate, len(input.Update))
		for i, u := range input.Update {
			u.ProductID = input.ProductID
			toUpdate[i] = u
		}
		if len(toAdd) > 0 {
			if _, err := steps.AddProductOptionsToProductStep.Run(e, toAdd); err != nil {
				return nil, err
			}
		}
		if len(toRemove) > 0 {
			if _, err := steps.RemoveProductOptionsFromProductStep.Run(e, toRemove); err != nil {
				return nil, err
			}
		}
		if len(toUpdate) > 0 {
			if _, err := steps.UpdateProductOptionValuesOnProductStep.Run(e, toUpdate); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

// LinkProductOptionsInput mirrors LinkProductOptionsToProductWorkflowInput:
// Add entries are option ids (strings), { id, value_ids } records or
// option creation records.
type LinkProductOptionsInput struct {
	ProductID string
	Add       []any
	Remove    []string
	Update    []services.ProductOptionProductValueUpdate
}

// CreateAndLinkProductOptionsToProductWorkflow mirrors
// createAndLinkProductOptionsToProductWorkflow.
var CreateAndLinkProductOptionsToProductWorkflow = composer.CreateWorkflow("create-and-link-product-options-to-product",
	func(e *composer.Execution, input LinkProductOptionsInput) (any, error) {
		var toCreate []*dal.Record
		var toAdd []string
		var toAddWithValues []services.ProductOptionProductPair
		for _, o := range input.Add {
			switch t := o.(type) {
			case string:
				toAdd = append(toAdd, t)
			case *dal.Record:
				if t.Has("id") && t.Has("value_ids") {
					id, _ := t.Get("id")
					vids, _ := t.Get("value_ids")
					l, _ := vids.([]any)
					toAddWithValues = append(toAddWithValues, services.ProductOptionProductPair{ProductOptionID: id.(string), ValueIDs: l, HasValueIDs: vids != nil})
				} else {
					toCreate = append(toCreate, t)
				}
			}
		}
		var createdIDs []any
		if len(toCreate) > 0 {
			created, err := steps.CreateProductOptionsStep.Run(e, toCreate)
			if err != nil {
				return nil, err
			}
			for _, c := range created {
				id, _ := c.Get("id")
				createdIDs = append(createdIDs, id)
			}
		}
		var add []services.ProductOptionProductPair
		for _, id := range toAdd {
			add = append(add, services.ProductOptionProductPair{ProductOptionID: id})
		}
		add = append(add, toAddWithValues...)
		for _, id := range createdIDs {
			add = append(add, services.ProductOptionProductPair{ProductOptionID: id.(string)})
		}
		if _, err := SetProductProductOptionsWorkflow.RunAsStep(e, SetProductProductOptionsInput{ProductID: input.ProductID, Add: add, Remove: input.Remove, Update: input.Update}); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("id", input.ProductID)
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.updated", Data: ev}); err != nil {
			return nil, err
		}
		return nil, nil
	})
