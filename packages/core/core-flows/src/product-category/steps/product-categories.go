// Package steps mirrors @medusajs/core-flows/src/product-category/steps.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

func product(c composer.Container) *services.ProductModuleService {
	s, _ := c.Resolve(utilsmodulessdk.PRODUCT).(*services.ProductModuleService)
	return s
}

// CreateProductCategoriesStep mirrors createProductCategoriesStep.
var CreateProductCategoriesStep = composer.CreateStep("create-product-categories",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := product(ctx.Container).CreateProductCategories(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		ids := make([]any, len(created))
		for i, c := range created {
			ids[i], _ = c.Get("id")
		}
		return composer.NewStepResponse(created, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return product(ctx.Container).DeleteProductCategories(ctx.Context, ids)
	})

// UpdateProductCategoriesStepInput mirrors { selector, update }.
type UpdateProductCategoriesStepInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

// UpdateProductCategoriesStep mirrors updateProductCategoriesStep.
var UpdateProductCategoriesStep = composer.CreateStep("update-product-categories",
	func(ctx composer.StepExecutionContext, in UpdateProductCategoriesStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := product(ctx.Container)
		var objs []common.ObjectLike
		if in.Update != nil {
			objs = []common.ObjectLike{in.Update}
		}
		selects, relations := common.GetSelectsAndRelationsFromObjectArray(objs, common.SelectsAndRelationsOptions{}, "")
		prev, err := svc.ListBySuffix(ctx.Context, "ProductCategories", in.Selector, utilsmodulessdk.FindConfig{Select: selects, Relations: relations})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := svc.UpdateProductCategories(ctx.Context, in.Selector, in.Update)
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
		_, err := product(ctx.Container).UpsertProductCategories(ctx.Context, prev)
		return err
	})

// DeleteProductCategoriesStep mirrors deleteProductCategoriesStep.
var DeleteProductCategoriesStep = composer.CreateStep("delete-product-categories",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		if err := product(ctx.Container).SoftDeleteProductCategories(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(ids)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		return product(ctx.Container).RestoreProductCategories(ctx.Context, ids)
	})
