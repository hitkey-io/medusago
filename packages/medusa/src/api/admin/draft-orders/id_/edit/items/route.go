// Package items mirrors api/admin/draft-orders/[id]/edit/items.
package items

import (
	dw "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	draftorders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/draft-orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/draft-orders/:id/edit/items.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	items := draftorders.BodyRecords(orders.BodyCopy(req), "items")
	preview, err := dw.AddDraftOrderItemsWorkflow.Run(req.Ctx(), req.Scope, dw.DraftOrderItemsInput{OrderID: req.Params["id"], Items: items})
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
