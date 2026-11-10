// Package shippingmethod mirrors api/admin/returns/[id]/shipping-method.
package shippingmethod

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
)

// POST mirrors POST /admin/returns/:id/shipping-method.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	input := orders.BodyCopy(req)
	input.Set("return_id", req.Params["id"])
	preview, err := workflows.CreateReturnShippingMethodWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	return returns.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
