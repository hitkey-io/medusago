package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateCollectionsInput mirrors CreateCollectionsWorkflowInput.
type CreateCollectionsInput struct {
	Collections    []*dal.Record
	AdditionalData *dal.Record
}

// CreateCollectionsWorkflow mirrors createCollectionsWorkflow.
var CreateCollectionsWorkflow = composer.CreateWorkflow("create-collections",
	func(e *composer.Execution, input CreateCollectionsInput) ([]*dal.Record, error) {
		created, err := steps.CreateCollectionsStep.Run(e, input.Collections)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-collection.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("collections", created)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "collectionsCreated", hookInput); err != nil {
			return nil, err
		}
		return created, nil
	})
