// Package shippingoptions mirrors api/admin/orders/[id]/shipping-options.
package shippingoptions

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/orders/:id/shipping-options.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	options, err := workflows.ListShippingOptionsForOrderWorkflow.Run(req.Ctx(), req.Scope, req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("shipping_options", options)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
