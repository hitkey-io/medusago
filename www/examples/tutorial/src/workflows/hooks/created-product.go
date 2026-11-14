// Package hooks consumes the hooks of Medusa's workflows.
package hooks

import (
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/modules/brand"
)

// region: register
// Register consumes the productsCreated hook of createProductsWorkflow.
// Call it once, before the application starts.
func Register() {
	workflows.CreateProductsWorkflow.RegisterHook("productsCreated", productsCreated, productsCreatedCompensation)
}

// endregion: register

// region: handler
func productsCreated(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
	input, _ := data.(*dal.Record)
	additionalData, _ := input.Get("additional_data")
	additional, _ := additionalData.(*dal.Record)
	if additional == nil {
		return composer.NewStepResponse[any](nil), nil
	}
	brandID, _ := additional.Get("brand_id")
	if brandID == nil {
		return composer.NewStepResponse[any](nil), nil
	}

	// if the brand doesn't exist, an error is thrown
	service := ctx.Container.Resolve(brand.ModuleName).(*brand.BrandModuleService)
	if _, err := service.Call(ctx.Context, "retrieveBrand", []any{brandID}); err != nil {
		return composer.StepResponse[any]{}, err
	}

	// endregion: handler
	// region: link
	var links []modulessdk.LinkDefinition
	createdProducts, _ := input.Get("products")
	products, _ := createdProducts.([]*dal.Record)
	for _, product := range products {
		productID, _ := product.Get("id")
		links = append(links, modulessdk.NewLinkDefinition(
			"product", "product_id", productID,
			brand.ModuleName, "brand_id", brandID,
		))
	}

	link := ctx.Container.Resolve(common.ContainerRegistrationKeys.LINK).(*modulessdk.Link)
	if _, err := link.Create(ctx.Context, links); err != nil {
		return composer.StepResponse[any]{}, err
	}

	logger := ctx.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	logger.Info("Linked brand to products")

	// the links are the input of the compensation function
	return composer.NewStepResponse[any](nil, links), nil
	// endregion: link
}

// region: compensation
func productsCreatedCompensation(ctx composer.StepExecutionContext, data any) error {
	links, _ := data.([]modulessdk.LinkDefinition)
	if len(links) == 0 {
		return nil
	}
	link := ctx.Container.Resolve(common.ContainerRegistrationKeys.LINK).(*modulessdk.Link)
	_, err := link.Dismiss(ctx.Context, links)
	return err
}

// endregion: compensation
