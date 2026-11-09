package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteProductTypesInput mirrors DeleteProductTypesWorkflowInput.
type DeleteProductTypesInput struct {
	IDs []any
}

// DeleteProductTypesWorkflow mirrors deleteProductTypesWorkflow.
var DeleteProductTypesWorkflow = composer.CreateWorkflow("delete-product-types",
	func(e *composer.Execution, input DeleteProductTypesInput) (any, error) {
		if _, err := steps.DeleteProductTypesStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("ids", input.IDs)
		if err := composer.CreateHook(e, "productTypesDeleted", hookInput); err != nil {
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
				_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{deleteInput(utilsmodulessdk.PRODUCT, "product_type_id", input.IDs)})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-type.deleted", Data: events})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
