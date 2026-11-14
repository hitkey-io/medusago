// Package nested holds the examples of the Execute Nested Workflows
// chapter.
package nested

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: run-as-step
// Workflow executes createProductsWorkflow as a step.
var Workflow = composer.CreateWorkflow("hello-world",
	func(e *composer.Execution, _ any) ([]*dal.Record, error) {
		product := dal.NewRecord()
		product.Set("title", "Shirt")
		// ...

		// the workflow's output is returned, not a result property
		products, err := workflows.CreateProductsWorkflow.RunAsStep(e, workflows.CreateProductsInput{
			Products: []*dal.Record{product},
		})
		if err != nil {
			return nil, err
		}

		// ...
		return products, nil
	})

// endregion: run-as-step

// region: input
// WorkflowInput is the input of the workflow.
type WorkflowInput struct {
	Title string `json:"title"`
}

// HelloProductWorkflow prepares the input of the nested workflow with
// regular Go code.
var HelloProductWorkflow = composer.CreateWorkflow("hello-product",
	func(e *composer.Execution, input WorkflowInput) ([]*dal.Record, error) {
		product := dal.NewRecord()
		product.Set("title", "Hello "+input.Title)

		return workflows.CreateProductsWorkflow.RunAsStep(e, workflows.CreateProductsInput{
			Products: []*dal.Record{product},
		})
	})

// endregion: input

// region: condition
// ConditionalInput is the input of the workflow.
type ConditionalInput struct {
	Product      *dal.Record
	ShouldCreate bool
}

// ConditionalWorkflow only runs createProductsWorkflow if ShouldCreate is
// enabled.
var ConditionalWorkflow = composer.CreateWorkflow("conditional-product",
	func(e *composer.Execution, input ConditionalInput) ([]*dal.Record, error) {
		if !input.ShouldCreate {
			return nil, nil
		}

		return workflows.CreateProductsWorkflow.RunAsStep(e, workflows.CreateProductsInput{
			Products: []*dal.Record{input.Product},
		})
	})

// endregion: condition
