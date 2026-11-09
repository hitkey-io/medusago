package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// UpdateCollectionsStep mirrors updateCollectionsStep.
var UpdateCollectionsStep = updateStep("update-collections", "ProductCollections",
	func(ctx context.Context, s *services.ProductModuleService, selector any, data *dal.Record) ([]*dal.Record, error) {
		return s.UpdateProductCollections(ctx, selector, data)
	},
	func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error) {
		return s.UpsertProductCollections(ctx, data)
	})
