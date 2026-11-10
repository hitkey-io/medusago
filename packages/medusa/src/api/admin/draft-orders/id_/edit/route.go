// Package edit mirrors api/admin/draft-orders/[id]/edit.
package edit

import (
	dw "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	draftorders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/draft-orders"
)

// POST mirrors POST /admin/draft-orders/:id/edit.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := dal.NewRecord()
	in.Set("order_id", req.Params["id"])
	preview, err := dw.BeginDraftOrderEditWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// DELETE mirrors DELETE /admin/draft-orders/:id/edit.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := dw.CancelDraftOrderEditWorkflow.Run(req.Ctx(), req.Scope, any(req.Params["id"])); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", req.Params["id"])
	out.Set("object", "draft-order-edit")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
