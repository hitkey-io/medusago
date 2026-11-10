// Package promotions mirrors api/admin/draft-orders/[id]/edit/promotions.
package promotions

import (
	dw "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	draftorders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/draft-orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/draft-orders/:id/edit/promotions.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	codes := draftorders.BodyStrings(orders.BodyCopy(req), "promo_codes")
	preview, err := dw.AddDraftOrderPromotionWorkflow.Run(req.Ctx(), req.Scope, dw.DraftOrderPromotionsInput{OrderID: req.Params["id"], PromoCodes: codes})
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// DELETE mirrors DELETE /admin/draft-orders/:id/edit/promotions.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	codes := draftorders.BodyStrings(orders.BodyCopy(req), "promo_codes")
	preview, err := dw.RemoveDraftOrderPromotionsWorkflow.Run(req.Ctx(), req.Scope, dw.DraftOrderPromotionsInput{OrderID: req.Params["id"], PromoCodes: codes})
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
