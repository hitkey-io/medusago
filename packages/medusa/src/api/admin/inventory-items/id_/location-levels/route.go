// Package locationlevels mirrors api/admin/inventory-items/[id]/location-levels/route.ts.
package locationlevels

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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
var Route = medusahttp.Route{GET: GET}
