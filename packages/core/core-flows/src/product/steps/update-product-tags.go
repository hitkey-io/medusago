package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// UpdateProductTagsStep mirrors updateProductTagsStep.
var UpdateProductTagsStep = updateStep("update-product-tags", "ProductTags",
	func(ctx context.Context, s *services.ProductModuleService, selector any, data *dal.Record) ([]*dal.Record, error) {
		return s.UpdateProductTags(ctx, selector, data)
	},
	func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error) {
		return s.UpsertProductTags(ctx, data)
	})
