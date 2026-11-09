package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductTypesInput mirrors UpdateProductTypesWorkflowInput.
type UpdateProductTypesInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	AdditionalData *dal.Record
}

// UpdateProductTypesWorkflow mirrors updateProductTypesWorkflow.
var UpdateProductTypesWorkflow = composer.CreateWorkflow("update-product-types",
	func(e *composer.Execution, input UpdateProductTypesInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateProductTypesStep.Run(e, steps.UpdateBySelectorInput{Selector: input.Selector, Update: input.Update})
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_types", updated)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productTypesUpdated", hookInput); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-type.updated", Data: idEvents(updated)}); err != nil {
			return nil, err
		}
		return updated, nil
	})
