// Package parallel holds the examples of the Run Workflow Steps in Parallel
// chapter.
package parallel

import (
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// WorkflowInput is the input of the workflow.
type WorkflowInput struct {
	Title string `json:"title"`
}

// Product is a product.
type Product struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

var createProductStep = composer.CreateStep("create-product",
	func(ctx composer.StepExecutionContext, input WorkflowInput) (composer.StepResponse[Product], error) {
		return composer.NewStepResponse(Product{ID: "prod_1", Title: input.Title}), nil
	}, nil)

var createPricesStep = composer.CreateStep("create-prices",
	func(ctx composer.StepExecutionContext, product Product) (composer.StepResponse[[]string], error) {
		return composer.NewStepResponse([]string{"price_1"}), nil
	}, nil)

var attachProductToSalesChannelStep = composer.CreateStep("attach-product-to-sales-channel",
	func(ctx composer.StepExecutionContext, product Product) (composer.StepResponse[string], error) {
		return composer.NewStepResponse("psc_1"), nil
	}, nil)

var getProductStep = composer.CreateStep("get-product",
	func(ctx composer.StepExecutionContext, id string) (composer.StepResponse[Product], error) {
		return composer.NewStepResponse(Product{ID: id, Title: "Shirt"}), nil
	}, nil)

// region: workflow
// MyWorkflow creates the product's prices and attaches it to a sales
// channel at the same point of the workflow, since neither step relies on
// the other's result.
var MyWorkflow = composer.CreateWorkflow("my-workflow",
	func(e *composer.Execution, input WorkflowInput) (Product, error) {
		product, err := createProductStep.Run(e, input)
		if err != nil {
			return Product{}, err
		}

		var (
			prices              []string
			productSalesChannel string
		)
		err = composer.Parallelize(
			func() (err error) {
				prices, err = createPricesStep.Run(e, product)
				return err
			},
			func() (err error) {
				productSalesChannel, err = attachProductToSalesChannelStep.Run(e, product)
				return err
			},
		)
		if err != nil {
			return Product{}, err
		}
		_, _ = prices, productSalesChannel

		return getProductStep.Run(e, product.ID)
	})

// endregion: workflow
