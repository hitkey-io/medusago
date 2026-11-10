// Package markasdelivered mirrors api/admin/orders/[id]/fulfillments/[fulfillment_id]/mark-as-delivered.
package markasdelivered

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/fulfillments/:fulfillment_id/mark-as-delivered.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body := orders.BodyCopy(req)
	nn, _ := body.Get("no_notification")
	if _, err := workflows.MarkOrderFulfillmentAsDeliveredWorkflow.Run(req.Ctx(), req.Scope, workflows.MarkOrderFulfillmentAsDeliveredInput{OrderID: req.Params["id"], FulfillmentID: req.Params["fulfillment_id"], NoNotification: nn}); err != nil {
		return err
	}
	order, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "order", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
