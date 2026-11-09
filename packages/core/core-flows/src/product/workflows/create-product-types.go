package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateProductTypesInput mirrors CreateProductTypesWorkflowInput.
type CreateProductTypesInput struct {
	ProductTypes   []*dal.Record
	AdditionalData *dal.Record
}

// CreateProductTypesWorkflow mirrors createProductTypesWorkflow.
var CreateProductTypesWorkflow = composer.CreateWorkflow("create-product-types",
	func(e *composer.Execution, input CreateProductTypesInput) ([]*dal.Record, error) {
		created, err := steps.CreateProductTypesStep.Run(e, input.ProductTypes)
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_types", created)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productTypesCreated", hookInput); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-type.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		return created, nil
	})
