// Package id_ mirrors api/admin/returns/[id].
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/returns/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v := dal.NewRecord()
	v.Set("id", req.Params["id"])
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			x, _ := req.FilterableFields.Get(k)
			filters.Set(k, x)
		}
	}
	v.Set("filters", filters)
	r, err := httputils.QueryOf(req.Scope).RemoteQuery(req.Ctx(), query.RemoteQueryObjectFromString(query.RemoteQueryObjectConfig{EntryPoint: "return", Variables: v, Fields: req.QueryConfig.Fields}), joiner.Options{ThrowIfKeyNotFound: true})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("return", r.Data[0])
	}
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
