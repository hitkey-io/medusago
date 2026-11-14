package compensation

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ERPService stands for the service of a module that integrates an ERP.
type ERPService interface {
	Retrieve(ctx context.Context, id string) (map[string]any, error)
	Create(ctx context.Context, data map[string]any) error
	Delete(ctx context.Context, id string) error
}

// DeleteItemsInput is the input of deleteItemsStep.
type DeleteItemsInput struct {
	IDs []string
}

// region: partial
// deleteItemsStep deletes items in the ERP. If deleting one of them fails,
// the step restores the items it deleted before returning the error, since
// the compensation function of a step that fails receives no data.
var deleteItemsStep = composer.CreateStep("delete-items-step",
	func(ctx composer.StepExecutionContext, input DeleteItemsInput) (composer.StepResponse[[]string], error) {
		erp := ctx.Container.Resolve("erp").(ERPService)
		var previous []map[string]any

		for _, id := range input.IDs {
			data, err := erp.Retrieve(ctx.Context, id)
			if err == nil {
				err = erp.Delete(ctx.Context, id)
			}
			if err != nil {
				// undo what the step did so far
				for _, item := range previous {
					_ = erp.Create(ctx.Context, item)
				}
				return composer.StepResponse[[]string]{}, err
			}

			previous = append(previous, data)
		}

		// the compensation function receives the deleted items
		return composer.NewStepResponse(input.IDs, previous), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		erp := ctx.Container.Resolve("erp").(ERPService)

		items, _ := data.([]map[string]any)
		for _, item := range items {
			if err := erp.Create(ctx.Context, item); err != nil {
				return err
			}
		}
		return nil
	})

// endregion: partial
