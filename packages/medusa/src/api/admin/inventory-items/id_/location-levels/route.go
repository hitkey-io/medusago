// Package locationlevels mirrors api/admin/inventory-items/[id]/location-levels/route.ts.
package locationlevels

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("inventory_item_id", req.Params["id"])
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "inventory_levels", httputils.ListVariables(filters, req.QueryConfig.Pagination), req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return res.Status(200).JSON(httputils.ListResponse("inventory_levels", r))
}

// Route is the route file.
// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	in.Set("inventory_item_id", req.Params["id"])
	if _, err := workflows.CreateInventoryLevelsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in}); err != nil {
		return err
	}
	return inventoryitems.Respond(req, res, req.Params["id"])
}

var Route = medusahttp.Route{GET: GET, POST: POST}
