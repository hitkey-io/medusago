// Package confirm mirrors api/admin/order-edits/[id]/confirm.
package confirm

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	orderedits "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/order-edits/:id/confirm.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := workflows.ConfirmOrderEditRequestWorkflow.Run(req.Ctx(), req.Scope, workflows.ConfirmOrderEditRequestInput{OrderID: req.Params["id"], ConfirmedBy: orders.ActorID(req)})
	if err != nil {
		return err
	}
	return orderedits.RespondPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
