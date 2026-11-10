// Package actionid mirrors api/admin/claims/[id]/inbound/items/[action_id].
package actionid

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/claims/:id/inbound/items/:action_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	returnID, err := claims.ClaimReturnID(req)
	if err != nil {
		return err
	}
	preview, err := workflows.UpdateRequestItemReturnWorkflow.Run(req.Ctx(), req.Scope, workflows.ReturnActionInput{ReturnID: returnID, ActionID: req.Params["action_id"], Data: orders.BodyCopy(req), ClaimID: req.Params["id"]})
	if err != nil {
		return err
	}
	return claims.RespondReturnDetails(req, res, preview, returnID)
}

// DELETE mirrors DELETE /admin/claims/:id/inbound/items/:action_id.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	returnID, err := claims.ClaimReturnID(req)
	if err != nil {
		return err
	}
	preview, err := workflows.RemoveItemReturnActionWorkflow.Run(req.Ctx(), req.Scope, workflows.ReturnActionInput{ReturnID: returnID, ActionID: req.Params["action_id"]})
	if err != nil {
		return err
	}
	return claims.RespondReturnDetails(req, res, preview, nil)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
