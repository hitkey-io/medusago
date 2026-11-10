// Package request mirrors api/admin/draft-orders/[id]/edit/request.
package request

import (
	dw "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	draftorders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/draft-orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/draft-orders/:id/edit/request.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	preview, err := dw.RequestDraftOrderEditWorkflow.Run(req.Ctx(), req.Scope, dw.DraftOrderEditUserInput{OrderID: req.Params["id"], UserID: orders.ActorID(req)})
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
