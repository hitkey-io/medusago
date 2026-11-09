package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteProductTagsInput mirrors DeleteProductTagsWorkflowInput.
type DeleteProductTagsInput struct {
	IDs []any
}

// DeleteProductTagsWorkflow mirrors deleteProductTagsWorkflow.
var DeleteProductTagsWorkflow = composer.CreateWorkflow("delete-product-tags",
	func(e *composer.Execution, input DeleteProductTagsInput) (any, error) {
		if _, err := steps.DeleteProductTagsStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("ids", input.IDs)
		if err := composer.CreateHook(e, "productTagsDeleted", hookInput); err != nil {
			return nil, err
		}
		events := make([]any, len(input.IDs))
		for i, id := range input.IDs {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-tag.deleted", Data: events}); err != nil {
			return nil, err
		}
		return nil, nil
	})
