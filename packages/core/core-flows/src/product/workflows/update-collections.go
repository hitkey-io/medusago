package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateCollectionsInput mirrors UpdateCollectionsWorkflowInput.
type UpdateCollectionsInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	AdditionalData *dal.Record
}

// UpdateCollectionsWorkflow mirrors updateCollectionsWorkflow.
var UpdateCollectionsWorkflow = composer.CreateWorkflow("update-collections",
	func(e *composer.Execution, input UpdateCollectionsInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateCollectionsStep.Run(e, steps.UpdateBySelectorInput{Selector: input.Selector, Update: input.Update})
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-collection.updated", Data: idEvents(updated)}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("collections", updated)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "collectionsUpdated", hookInput); err != nil {
			return nil, err
		}
		return updated, nil
	})
