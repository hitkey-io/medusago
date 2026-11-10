// Package cancel mirrors api/admin/orders/[id]/cancel.
package cancel

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/cancel.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := workflows.CancelOrderWorkflow.Run(req.Ctx(), req.Scope, struct {
		OrderID    any
		CanceledBy any
	}{req.Params["id"], orders.ActorID(req)}); err != nil {
		return err
	}
	return orders.RespondOrder(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
