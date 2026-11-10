// Package shipments mirrors api/admin/orders/[id]/fulfillments/[fulfillment_id]/shipments.
package shipments

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/fulfillments/:fulfillment_id/shipments.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("order_id", req.Params["id"])
	in.Set("fulfillment_id", req.Params["fulfillment_id"])
	if v, ok := in.Get("labels"); !ok || v == nil {
		in.Set("labels", []any{})
	}
	in.Set("created_by", orders.ActorID(req))
	if _, err := workflows.CreateOrderShipmentWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateOrderShipmentInput{OrderID: req.Params["id"], FulfillmentID: req.Params["fulfillment_id"], Input: in}); err != nil {
		return err
	}
	return orders.RespondOrder(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
