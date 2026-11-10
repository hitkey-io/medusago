// Package id_ mirrors api/store/product-categories/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
)

// GET mirrors GET (upstream checks the data array, so a missing category
// is answered with {}).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			f.Set(k, v)
		}
	}
	r, err := storeutils.Graph(req.Ctx(), req.Scope, query.GraphInput{Entity: "product_category", Fields: req.QueryConfig.Fields, Filters: f, Context: storeutils.PublishedProductsContext()}, req.Locale)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("product_category", r.Data[0])
	}
	return res.JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
