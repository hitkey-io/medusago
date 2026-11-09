// Package id_ mirrors api/admin/stock-locations/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/stock-location/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	stocklocations "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/stock-locations"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	if _, err := workflows.UpdateStockLocationsWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateStockLocationsInput{Selector: selector, Update: body}); err != nil {
		return err
	}
	return stocklocations.Respond(req, res, req.Params["id"])
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteStockLocationsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "stock_location")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("stock_location", "stock_location", "Stock location with id: %s was not found"), POST: POST, DELETE: DELETE}
