package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/product/src/services"
)

// CreateCollectionsStep mirrors createCollectionsStep.
var CreateCollectionsStep = createStep("create-collections", "ProductCollection",
	func(ctx context.Context, s *services.ProductModuleService, data []*dal.Record) ([]*dal.Record, error) {
		return s.CreateProductCollections(ctx, data)
	})
