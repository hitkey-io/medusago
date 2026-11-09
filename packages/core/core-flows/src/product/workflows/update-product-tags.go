package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductTagsInput mirrors UpdateProductTagsWorkflowInput.
type UpdateProductTagsInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	AdditionalData *dal.Record
}

// UpdateProductTagsWorkflow mirrors updateProductTagsWorkflow.
var UpdateProductTagsWorkflow = composer.CreateWorkflow("update-product-tags",
	func(e *composer.Execution, input UpdateProductTagsInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateProductTagsStep.Run(e, steps.UpdateBySelectorInput{Selector: input.Selector, Update: input.Update})
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_tags", updated)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productTagsUpdated", hookInput); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-tag.updated", Data: idEvents(updated)}); err != nil {
			return nil, err
		}
		return updated, nil
	})
