package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteProductOptionsInput mirrors DeleteProductOptionsWorkflowInput.
type DeleteProductOptionsInput struct {
	IDs []any
}

// DeleteProductOptionsWorkflow mirrors deleteProductOptionsWorkflow.
var DeleteProductOptionsWorkflow = composer.CreateWorkflow("delete-product-options",
	func(e *composer.Execution, input DeleteProductOptionsInput) (any, error) {
		if _, err := steps.DeleteProductOptionsStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("ids", input.IDs)
		if err := composer.CreateHook(e, "productOptionsDeleted", hookInput); err != nil {
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
				_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{deleteInput(utilsmodulessdk.PRODUCT, "product_option_id", input.IDs)})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-option.deleted", Data: events})
				return err
			},
		)
		return nil, err
	})
