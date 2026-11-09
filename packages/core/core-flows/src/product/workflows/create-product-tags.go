package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateProductTagsInput mirrors CreateProductTagsWorkflowInput.
type CreateProductTagsInput struct {
	ProductTags    []*dal.Record
	AdditionalData *dal.Record
}

// CreateProductTagsWorkflow mirrors createProductTagsWorkflow.
var CreateProductTagsWorkflow = composer.CreateWorkflow("create-product-tags",
	func(e *composer.Execution, input CreateProductTagsInput) ([]*dal.Record, error) {
		created, err := steps.CreateProductTagsStep.Run(e, input.ProductTags)
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("product_tags", created)
		hookInput.Set("additional_data", input.AdditionalData)
		if err := composer.CreateHook(e, "productTagsCreated", hookInput); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "product-tag.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		return created, nil
	})
