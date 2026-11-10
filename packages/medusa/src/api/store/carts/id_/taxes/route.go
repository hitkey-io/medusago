// Package taxes mirrors api/store/carts/[id]/taxes.
package taxes

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

// POST mirrors POST /store/carts/:id/taxes.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := cartworkflows.UpdateTaxLinesWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.UpdateTaxLinesInput{CartID: req.Params["id"], ForceTaxCalculation: true}); err != nil {
		return err
	}
	return carts.RespondCart(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
