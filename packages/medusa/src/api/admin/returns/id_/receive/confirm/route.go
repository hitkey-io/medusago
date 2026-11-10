// Package confirm mirrors api/admin/returns/[id]/receive/confirm.
package confirm

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
)

// POST mirrors POST /admin/returns/:id/receive/confirm.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.ConfirmReturnReceiveWorkflow.Run(req.Ctx(), req.Scope, workflows.ConfirmReturnRequestInput{ReturnID: req.Params["id"], ConfirmedBy: orders.ActorID(req)})
	if err != nil {
		return err
	}
	return returns.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
