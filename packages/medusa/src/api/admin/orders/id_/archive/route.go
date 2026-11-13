// Package archive mirrors api/admin/orders/[id]/archive.
package archive

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/archive.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := workflows.ArchiveOrderWorkflow.Run(req.Ctx(), req.Scope, workflows.ArchiveOrdersInput{OrderIDs: []any{req.Params["id"]}}); err != nil {
		return err
	}
	return orders.RespondOrder(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
