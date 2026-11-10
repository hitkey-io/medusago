package products

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET: with the index engine, the filtered listings go
// through query.index (getProductsWithIndexEngine).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if indexEngineEnabled(req) && useIndexEngine(req) {
		return getProductsWithIndexEngine(req, res)
	}
	return getProducts(req, res)
}

// getProducts mirrors getProducts.
func getProducts(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
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

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	product := dal.NewRecord()
	var additional *dal.Record
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k == "additional_data" {
			additional, _ = v.(*dal.Record)
			continue
		}
		product.Set(k, v)
	}
	result, err := workflows.CreateProductsWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateProductsInput{Products: []*dal.Record{product}, AdditionalData: additional})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	p, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product", id, RemapKeysForProduct(req.QueryConfig.Fields))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("product", RemapProductResponse(p))
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
