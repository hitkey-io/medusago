package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// UpdateProductTypesStep mirrors updateProductTypesStep.
var UpdateProductTypesStep = updateStep("update-product-types", "ProductTypes",
	func(ctx context.Context, s *services.ProductModuleService, selector any, data *dal.Record) ([]*dal.Record, error) {
		return s.UpdateProductTypes(ctx, selector, data)
	},
	func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error) {
		return s.UpsertProductTypes(ctx, data)
	})
