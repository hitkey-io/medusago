// Package shipment mirrors api/admin/fulfillments/[id]/shipment.
package shipment

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillments"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/fulfillments/:id/shipment.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("id", req.Params["id"])
	in.Set("marked_shipped_by", orders.ActorID(req))
	if _, err := workflows.CreateShipmentWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return fulfillments.RespondFulfillment(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
