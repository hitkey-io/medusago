// Package shippingmethod mirrors api/admin/claims/[id]/inbound/shipping-method.
package shippingmethod

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/claims/:id/inbound/shipping-method.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	returnID, err := claims.ClaimReturnID(req)
	if err != nil {
		return err
	}
	input := orders.BodyCopy(req)
	input.Set("return_id", returnID)
	input.Set("claim_id", req.Params["id"])
	preview, err := workflows.CreateClaimShippingMethodWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	return claims.RespondReturnPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
