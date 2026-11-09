package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateProductVariantsStepInput mirrors UpdateProductVariantsStepInput.
type UpdateProductVariantsStepInput struct {
	Batch           bool
	ProductVariants []*dal.Record
	Selector        *dal.Record
	Update          *dal.Record
}

// UpdateProductVariantsStep mirrors updateProductVariantsStep.
var UpdateProductVariantsStep = composer.CreateStep("update-product-variants",
	func(ctx composer.StepExecutionContext, in UpdateProductVariantsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := ProductService(ctx.Container)
		if in.Batch {
			for _, v := range in.ProductVariants {
				if id, _ := v.Get("id"); id == nil || id == "" {
					return composer.StepResponse[[]*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Product variant ID is required when doing a batch update of product variants")
				}
			}
			if len(in.ProductVariants) == 0 {
				return composer.NewStepResponse([]*dal.Record{}, any([]*dal.Record{})), nil
			}
			f := dal.NewRecord()
			f.Set("id", ids(in.ProductVariants))
			prev, err := svc.ListBySuffix(ctx.Context, "ProductVariants", f, utilsmodulessdk.FindConfig{})
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			out, err := svc.UpsertProductVariants(ctx.Context, in.ProductVariants)
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			return composer.NewStepResponse(out, any(prev)), nil
		}
		prev, err := prevDataFor(ctx.Context, svc, "ProductVariants", UpdateBySelectorInput{Selector: in.Selector, Update: in.Update})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateProductVariants(ctx.Context, in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := ProductService(ctx.Container).UpsertProductVariants(ctx.Context, prev)
		return err
	})

// DeleteProductVariantsStep mirrors deleteProductVariantsStep.
var DeleteProductVariantsStep = softDeleteStep("delete-product-variants", "ProductVariant")
