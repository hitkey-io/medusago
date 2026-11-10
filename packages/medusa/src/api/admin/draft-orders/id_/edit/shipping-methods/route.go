// Package shippingmethods mirrors api/admin/draft-orders/[id]/edit/shipping-methods.
package shippingmethods

import (
	dw "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	draftorders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/draft-orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/draft-orders/:id/edit/shipping-methods.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("order_id", req.Params["id"])
	preview, err := dw.AddDraftOrderShippingMethodsWorkflow.Run(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
