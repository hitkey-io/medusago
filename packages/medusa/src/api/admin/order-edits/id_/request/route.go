// Package request mirrors api/admin/order-edits/[id]/request.
package request

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	orderedits "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/order-edits/:id/request.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	nn, _ := orders.BodyCopy(req).Get("no_notification")
	preview, err := workflows.RequestOrderEditRequestWorkflow.Run(req.Ctx(), req.Scope, workflows.OrderEditRequestInput{OrderID: req.Params["id"], RequestedBy: orders.ActorID(req), NoNotification: nn})
	if err != nil {
		return err
	}
	return orderedits.RespondPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
