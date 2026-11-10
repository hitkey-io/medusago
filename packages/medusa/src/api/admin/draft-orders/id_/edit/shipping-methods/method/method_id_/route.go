// Package methodid mirrors api/admin/draft-orders/[id]/edit/shipping-methods/method/[method_id].
package methodid

import (
	dw "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	draftorders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/draft-orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/draft-orders/:id/edit/shipping-methods/method/:method_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	data := orders.BodyCopy(req)
	d := dal.NewRecord()
	d.Set("shipping_method_id", req.Params["method_id"])
	for _, k := range data.Keys() {
		v, _ := data.Get(k)
		d.Set(k, v)
	}
	preview, err := dw.UpdateDraftOrderShippingMethodWorkflow.Run(req.Ctx(), req.Scope, dw.DraftOrderActionInput{OrderID: req.Params["id"], Data: d})
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// DELETE mirrors DELETE /admin/draft-orders/:id/edit/shipping-methods/method/:method_id.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	d := dal.NewRecord()
	d.Set("shipping_method_id", req.Params["method_id"])
	preview, err := dw.RemoveDraftOrderShippingMethodWorkflow.Run(req.Ctx(), req.Scope, dw.DraftOrderActionInput{OrderID: req.Params["id"], Data: d})
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
