package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// CreateProductTagsStep mirrors createProductTagsStep.
var CreateProductTagsStep = createStep("create-product-tags", "ProductTag",
	func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error) {
		return s.CreateProductTags(ctx, data)
	})
