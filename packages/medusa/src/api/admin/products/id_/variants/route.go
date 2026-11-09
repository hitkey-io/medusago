// Package variants mirrors api/admin/products/[id]/variants/route.ts.
package variants

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
	mwproducts "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/middlewares/products"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	fields, withInventoryQuantity := mwproducts.PrepareInventoryQuantityFields(req.QueryConfig.Fields, "")
	req.QueryConfig.Fields = fields
	filters := dal.NewRecord()
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	filters.Set("product_id", req.Params["id"])
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "variant", IDOrFilter: filters, Fields: products.RemapKeysForVariant(req.QueryConfig.Fields), Pagination: req.QueryConfig.Pagination,
	})
	if err != nil {
		return err
	}
	if withInventoryQuantity {
		if err := mwproducts.WrapVariantsWithTotalInventoryQuantity(req.Ctx(), httputils.QueryOf(req.Scope), r.Data); err != nil {
			return err
		}
	}
	vs := make([]any, len(r.Data))
	for i, v := range r.Data {
		vs[i] = products.RemapVariantResponse(v)
	}
	r.Data = vs
	return res.JSON(httputils.ListResponse("variants", r))
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
