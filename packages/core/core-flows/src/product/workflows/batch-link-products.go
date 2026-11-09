package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func linkEvents(in steps.LinkWorkflowInput) []any {
	seen := map[string]bool{}
	out := []any{}
	for _, id := range append(append([]any{}, in.Add...), in.Remove...) {
		if seen[fmt.Sprint(id)] {
			continue
		}
		seen[fmt.Sprint(id)] = true
		ev := dal.NewRecord()
		ev.Set("id", id)
		out = append(out, ev)
	}
	return out
}

// BatchLinkProductsToCategoryWorkflow mirrors batchLinkProductsToCategoryWorkflow.
var BatchLinkProductsToCategoryWorkflow = composer.CreateWorkflow("batch-link-products-to-category",
	func(e *composer.Execution, input steps.LinkWorkflowInput) (any, error) {
		if _, err := steps.BatchLinkProductsToCategoryStep.Run(e, input); err != nil {
			return nil, err
		}
		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.updated", Data: linkEvents(input)})
		return nil, err
	})

// BatchLinkProductsToCollectionWorkflow mirrors batchLinkProductsToCollectionWorkflow.
var BatchLinkProductsToCollectionWorkflow = composer.CreateWorkflow("batch-link-products-to-collection",
	func(e *composer.Execution, input steps.LinkWorkflowInput) (any, error) {
		if _, err := steps.BatchLinkProductsToCollectionStep.Run(e, input); err != nil {
			return nil, err
		}
		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product.updated", Data: linkEvents(input)})
		return nil, err
	})
