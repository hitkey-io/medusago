// Package actionid mirrors api/admin/order-edits/[id]/items/[action_id].
package actionid

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	orderedits "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/order-edits/:id/items/:action_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.UpdateOrderEditAddItemWorkflow.Run(req.Ctx(), req.Scope, workflows.OrderChangeActionInput{OrderID: req.Params["id"], ActionID: req.Params["action_id"], Data: orders.BodyCopy(req)})
	if err != nil {
		return err
	}
	return orderedits.RespondPreview(res, preview)
}

// DELETE mirrors DELETE /admin/order-edits/:id/items/:action_id.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.RemoveItemOrderEditActionWorkflow.Run(req.Ctx(), req.Scope, workflows.OrderChangeActionInput{OrderID: req.Params["id"], ActionID: req.Params["action_id"]})
	if err != nil {
		return err
	}
	return orderedits.RespondPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
