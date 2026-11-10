// Package request mirrors api/admin/returns/[id]/request.
package request

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
)

// POST mirrors POST /admin/returns/:id/request.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.ConfirmReturnRequestWorkflow.Run(req.Ctx(), req.Scope, workflows.ConfirmReturnRequestInput{ReturnID: req.Params["id"], ConfirmedBy: orders.ActorID(req)})
	if err != nil {
		return err
	}
	return returns.RespondPreview(req, res, preview)
}

// DELETE mirrors DELETE /admin/returns/:id/request.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := workflows.CancelReturnRequestWorkflow.Run(req.Ctx(), req.Scope, any(req.Params["id"])); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", req.Params["id"])
	out.Set("object", "return")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
