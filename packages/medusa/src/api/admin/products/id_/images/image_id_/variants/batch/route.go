// Package batch mirrors api/admin/products/[id]/images/[image_id]/variants/batch/route.ts.
package batch

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

// Route is the route file.
var Route = medusahttp.Route{POST: products.BatchImagesRoute("image_id", func(ctx context.Context, scope medusahttp.MedusaContainer, id any, add, remove []any) ([]any, []any, error) {
	r, err := workflows.BatchImageVariantsWorkflow.Run(ctx, scope, workflows.BatchImagesInput{ID: id, Add: add, Remove: remove})
	return r.Added, r.Removed, err
})}
