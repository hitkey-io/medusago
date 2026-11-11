package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteCollectionsInput mirrors DeleteCollectionsWorkflowInput.
type DeleteCollectionsInput struct {
	IDs []any
}

// DeleteCollectionsWorkflow mirrors deleteCollectionsWorkflow.
var DeleteCollectionsWorkflow = composer.CreateWorkflow("delete-collections",
	func(e *composer.Execution, input DeleteCollectionsInput) (any, error) {
		if _, err := steps.DeleteCollectionsStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		events := make([]any, len(input.IDs))
		for i, id := range input.IDs {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		err := composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-collection.deleted", Data: events})
				return err
			},
			func() error {
				_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{deleteInput(utilsmodulessdk.PRODUCT, "product_collection_id", input.IDs)})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("ids", input.IDs)
		if err := composer.CreateHook(e, "collectionsDeleted", hookInput); err != nil {
			return nil, err
		}
		return nil, nil
	})
