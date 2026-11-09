// Package fulfillmentsets mirrors api/admin/stock-locations/[id]/fulfillment-sets/route.ts.
package fulfillmentsets

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/stock-location/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	stocklocations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	name, _ := body.Get("name")
	typ, _ := body.Get("type")
	if _, err := workflows.CreateLocationFulfillmentSetWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateLocationFulfillmentSetInput{LocationID: req.Params["id"], Name: name, Type: typ}); err != nil {
		return err
	}
	return stocklocations.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
