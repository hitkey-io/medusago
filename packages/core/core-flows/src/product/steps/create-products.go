// Package steps mirrors @medusajs/core-flows/src/product/steps.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// ProductService resolves the product module service.
func ProductService(c composer.Container) *services.ProductModuleService {
	s, _ := c.Resolve(utilsmodulessdk.PRODUCT).(*services.ProductModuleService)
	return s
}

func ids(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i], _ = r.Get("id")
	}
	return out
}

// CreateProductsStep mirrors createProductsStep.
var CreateProductsStep = composer.CreateStep("create-products",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := ProductService(ctx.Container).CreateProducts(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		idList, _ := data.([]any)
		if len(idList) == 0 {
			return nil
		}
		_, err := ProductService(ctx.Container).Internal("Product").Delete(ctx.Context, idList)
		return err
	})

// CreateProductVariantsStep mirrors createProductVariantsStep.
var CreateProductVariantsStep = composer.CreateStep("create-product-variants",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := ProductService(ctx.Container).CreateProductVariants(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(ids(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		idList, _ := data.([]any)
		if len(idList) == 0 {
			return nil
		}
		_, err := ProductService(ctx.Container).Internal("ProductVariant").Delete(ctx.Context, idList)
		return err
	})

// VariantPriceSetLink is { variant_id, price_set_id }.
type VariantPriceSetLink struct {
	VariantID  any
	PriceSetID any
}

func pricingLinks(links []VariantPriceSetLink) []*dal.Record {
	out := make([]*dal.Record, len(links))
	for i, l := range links {
		out[i] = modulessdk.NewLinkDefinition("product", "variant_id", l.VariantID, "pricing", "price_set_id", l.PriceSetID)
	}
	return out
}

// CreateVariantPricingLinkStep mirrors createVariantPricingLinkStep.
var CreateVariantPricingLinkStep = composer.CreateStep("create-variant-pricing-link",
	func(ctx composer.StepExecutionContext, links []VariantPriceSetLink) (composer.StepResponse[any], error) {
		if _, err := steps.LinkOf(ctx.Container).Create(ctx.Context, pricingLinks(links)); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(links)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		links, _ := data.([]VariantPriceSetLink)
		if len(links) == 0 {
			return nil
		}
		_, err := steps.LinkOf(ctx.Container).Dismiss(ctx.Context, pricingLinks(links))
		return err
	})
