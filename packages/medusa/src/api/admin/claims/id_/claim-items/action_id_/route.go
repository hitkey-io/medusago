// Package actionid mirrors api/admin/claims/[id]/claim-items/[action_id].
package actionid

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/claims/:id/claim-items/:action_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.UpdateClaimItemWorkflow.Run(req.Ctx(), req.Scope, workflows.RelatedActionInput{ID: req.Params["id"], ActionID: req.Params["action_id"], Data: orders.BodyCopy(req)})
	if err != nil {
		return err
	}
	return claims.RespondPreview(req, res, preview)
}

// DELETE mirrors DELETE /admin/claims/:id/claim-items/:action_id.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.RemoveItemClaimActionWorkflow.Run(req.Ctx(), req.Scope, workflows.RelatedActionInput{ID: req.Params["id"], ActionID: req.Params["action_id"]})
	if err != nil {
		return err
	}
	return claims.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
