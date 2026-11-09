// Package servicezones mirrors api/admin/fulfillment-sets/[id]/service-zones/route.ts.
package servicezones

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	fulfillmentsets "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillment-sets"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	in.Set("fulfillment_set_id", req.Params["id"])
	name, _ := body.Get("name")
	in.Set("name", name)
	if gz, ok := body.Get("geo_zones"); ok {
		in.Set("geo_zones", gz)
	}
	if _, err := workflows.CreateServiceZonesWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in}); err != nil {
		return err
	}
	v, err := fulfillmentsets.RefetchFulfillmentSet(req, req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if v != nil {
		out.Set("fulfillment_set", v)
	}
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
