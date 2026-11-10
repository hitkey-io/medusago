// Package shippingmethod mirrors api/admin/exchanges/[id]/outbound/shipping-method.
package shippingmethod

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/exchanges/:id/outbound/shipping-method.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	input := orders.BodyCopy(req)
	input.Set("exchange_id", req.Params["id"])
	preview, err := workflows.CreateExchangeShippingMethodWorkflow.Run(req.Ctx(), req.Scope, input)
	if err != nil {
		return err
	}
	return exchanges.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
