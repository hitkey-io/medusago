// Package complete mirrors api/admin/orders/[id]/complete.
package complete

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/complete.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	ad, _ := orders.BodyCopy(req).Get("additional_data")
	if _, err := workflows.CompleteOrderWorkflow.Run(req.Ctx(), req.Scope, workflows.CompleteOrdersInput{OrderIDs: []any{req.Params["id"]}, AdditionalData: ad}); err != nil {
		return err
	}
	return orders.RespondOrder(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
