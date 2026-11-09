// Package values mirrors api/admin/product-options/[id]/values/route.ts.
package values

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
	filters.Set("option_id", req.Params["id"])
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{Entity: "product_option_value", IDOrFilter: filters, Fields: req.QueryConfig.Fields, Pagination: req.QueryConfig.Pagination})
	if err != nil {
		return err
	}
	return res.JSON(httputils.ListResponse("product_option_values", r))
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
