package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductOptionValuesStepInput mirrors { id, update }.
type UpdateProductOptionValuesStepInput struct {
	ID     string
	Update *dal.Record
}

// UpdateProductOptionValuesStep mirrors updateProductOptionValuesStep.
var UpdateProductOptionValuesStep = composer.CreateStep("update-product-option-values",
	func(ctx composer.StepExecutionContext, in UpdateProductOptionValuesStepInput) (composer.StepResponse[*dal.Record], error) {
		svc := ProductService(ctx.Container)
		prev, err := svc.Internal("ProductOptionValue").Retrieve(ctx.Context, in.ID, utilsmodulessdk.FindConfig{Select: []string{"id", "value", "metadata"}})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		out, err := svc.UpdateProductOptionValues(ctx.Context, in.ID, in.Update)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.(*dal.Record)
		if prev == nil {
			return nil
		}
		id, _ := prev.Get("id")
		u := dal.NewRecord()
		v, _ := prev.Get("value")
		m, _ := prev.Get("metadata")
		u.Set("value", v)
		u.Set("metadata", m)
		_, err := ProductService(ctx.Container).UpdateProductOptionValues(ctx.Context, id.(string), u)
		return err
	})

// DeleteProductOptionValuesStep mirrors deleteProductOptionValuesStep.
var DeleteProductOptionValuesStep = composer.CreateStep("delete-product-option-values",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if err := ProductService(ctx.Container).SoftDeleteProductOptionValues(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := ProductService(ctx.Container).Internal("ProductOptionValue").Restore(ctx.Context, ids)
		return err
	})
