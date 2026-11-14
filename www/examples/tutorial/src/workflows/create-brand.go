// Package workflows holds the workflows of the project.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/modules/brand"
)

// region: step-input
// CreateBrandStepInput is the input of createBrandStep.
type CreateBrandStepInput struct {
	Name string `json:"name"`
}

// endregion: step-input

// region: helper
// brandService resolves the Brand Module's service from the container.
func brandService(c composer.Container) *brand.BrandModuleService {
	return c.Resolve(brand.ModuleName).(*brand.BrandModuleService)
}

// endregion: helper

// region: step
// CreateBrandStep creates a brand. Its compensation function deletes it.
var CreateBrandStep = composer.CreateStep("create-brand-step",
	func(ctx composer.StepExecutionContext, input CreateBrandStepInput) (composer.StepResponse[*dal.Record], error) {
		data := dal.NewRecord()
		data.Set("name", input.Name)

		created, err := brandService(ctx.Container).Call(ctx.Context, "createBrands", []any{data})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}

		// the brand is the step's output, its id the input of the
		// compensation function
		record := created.(*dal.Record)
		id, _ := record.Get("id")
		return composer.NewStepResponse(record, id), nil
	},
	// region: compensation
	func(ctx composer.StepExecutionContext, id any) error {
		if id == nil {
			return nil
		}
		_, err := brandService(ctx.Container).Call(ctx.Context, "deleteBrands", []any{id})
		return err
	},
	// endregion: compensation
)

// endregion: step

// region: workflow
// CreateBrandWorkflowInput is the input of createBrandWorkflow.
type CreateBrandWorkflowInput struct {
	Name string `json:"name"`
}

// CreateBrandWorkflow creates a brand.
var CreateBrandWorkflow = composer.CreateWorkflow("create-brand",
	func(e *composer.Execution, input CreateBrandWorkflowInput) (*dal.Record, error) {
		return CreateBrandStep.Run(e, CreateBrandStepInput(input))
	})

// endregion: workflow
