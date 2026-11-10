// Package shippingmethod mirrors api/admin/order-edits/[id]/shipping-method.
package shippingmethod

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	orderedits "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/order-edits/:id/shipping-method.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	input := orders.BodyCopy(req)
	input.Set("order_id", req.Params["id"])
	preview, err := workflows.CreateOrderEditShippingMethodWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	return orderedits.RespondPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
