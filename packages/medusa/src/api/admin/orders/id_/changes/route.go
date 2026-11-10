// Package changes mirrors api/admin/orders/[id]/changes.
package changes

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/orders/:id/changes.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("order_id", req.Params["id"])
	v := dal.NewRecord()
	v.Set("filters", filters)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "order_change", v, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	data := r.Data
	if data == nil {
		data = []any{}
	}
	out.Set("order_changes", data)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
