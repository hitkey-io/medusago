package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// GET mirrors GET (index engine disabled: getProducts).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	withDeleted := req.QueryConfig.WithDeleted != nil && *req.QueryConfig.WithDeleted
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "product", IDOrFilter: req.FilterableFields, Fields: RemapKeysForProduct(req.QueryConfig.Fields),
		Pagination: req.QueryConfig.Pagination, WithDeleted: withDeleted,
	})
	if err != nil {
		return err
	}
	products := make([]any, len(r.Data))
	for i, p := range r.Data {
		products[i] = RemapProductResponse(p)
	}
	r.Data = products
	return res.JSON(httputils.ListResponse("products", r))
}

// Route is route.ts (POST needs createProductsWorkflow).
var Route = medusahttp.Route{GET: GET}
