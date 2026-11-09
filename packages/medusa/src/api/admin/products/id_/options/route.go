// Package options mirrors api/admin/products/[id]/options/route.ts.
package options

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
	p := dal.NewRecord()
	p.Set("id", req.Params["id"])
	filters.Set("products", p)
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "product_option", IDOrFilter: filters, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination})
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("product_options", r))
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
