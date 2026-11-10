// Package actionid mirrors api/admin/returns/[id]/receive-items/[action_id].
package actionid

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
)

// POST mirrors POST /admin/returns/:id/id_/receive-items/:action_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.UpdateReceiveItemReturnRequestWorkflow.Run(req.Ctx(), req.Scope, workflows.ReturnActionInput{ReturnID: req.Params["id"], ActionID: req.Params["action_id"], Data: orders.BodyCopy(req)})
	if err != nil {
		return err
	}
	return returns.RespondPreview(req, res, preview)
}

// DELETE mirrors DELETE /admin/returns/:id/id_/receive-items/:action_id.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.RemoveItemReceiveReturnActionWorkflow.Run(req.Ctx(), req.Scope, workflows.ReturnActionInput{ReturnID: req.Params["id"], ActionID: req.Params["action_id"]})
	if err != nil {
		return err
	}
	return returns.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
