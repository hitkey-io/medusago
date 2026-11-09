// Package options mirrors api/admin/fulfillment-providers/[id]/options/route.ts.
package options

import (
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	opts, err := fulfillmentsteps.Fulfillment(req.Scope).RetrieveFulfillmentOptions(req.Ctx(), req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("fulfillment_options", opts)
	out.Set("count", len(opts))
	out.Set("limit", len(opts))
	out.Set("offset", 0)
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
