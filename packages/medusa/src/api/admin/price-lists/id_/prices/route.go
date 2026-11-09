// Package prices mirrors api/admin/price-lists/[id]/prices/route.ts.
package prices

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			f.Set(k, v)
		}
	}
	f.Set("price_list_id", req.Params["id"])
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "price", Fields: req.QueryConfig.Fields, Filters: f, Pagination: req.QueryConfig.Pagination}, joiner.Options{})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	data := r.Data
	if data == nil {
		data = []any{}
	}
	out.Set("prices", data)
	meta := func(k string) any {
		if r.Metadata != nil {
			if v, ok := r.Metadata.Get(k); ok && v != nil {
				return v
			}
		}
		return float64(0)
	}
	out.Set("count", meta("count"))
	out.Set("offset", meta("skip"))
	out.Set("limit", meta("take"))
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
