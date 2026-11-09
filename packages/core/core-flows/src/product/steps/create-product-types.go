package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// CreateProductTypesStep mirrors createProductTypesStep.
var CreateProductTypesStep = createStep("create-product-types", "ProductType",
	func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error) {
		return s.CreateProductTypes(ctx, data)
	})
