package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductOptionValuesWorkflow mirrors updateProductOptionValuesWorkflow.
var UpdateProductOptionValuesWorkflow = composer.CreateWorkflow("update-product-option-values",
	func(e *composer.Execution, input steps.UpdateProductOptionValuesStepInput) (*dal.Record, error) {
		updated, err := steps.UpdateProductOptionValuesStep.Run(e, input)
		if err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("id", input.ID)
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-option-value.updated", Data: ev}); err != nil {
			return nil, err
		}
		return updated, nil
	})

// DeleteProductOptionValuesWorkflow mirrors deleteProductOptionValuesWorkflow.
var DeleteProductOptionValuesWorkflow = composer.CreateWorkflow("delete-product-option-values",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteProductOptionValuesStep.Run(e, ids); err != nil {
			return nil, err
		}
		events := make([]any, len(ids))
		for i, id := range ids {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-option-value.deleted", Data: events}); err != nil {
			return nil, err
		}
		return nil, nil
	})
