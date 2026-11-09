// Package variants mirrors api/admin/products/[id]/variants/route.ts.
package variants

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
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

// splitAdditional returns the body without additional_data, and
// additional_data.
func splitAdditional(body *dal.Record) (*dal.Record, *dal.Record) {
	rest := dal.NewRecord()
	var additional *dal.Record
	if body == nil {
		return rest, nil
	}
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k == "additional_data" {
			additional, _ = v.(*dal.Record)
			continue
		}
		rest.Set(k, v)
	}
	return rest, additional
}

// RespondProduct refetches the product and writes { product }.
func RespondProduct(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, productID any) error {
	p, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "product", productID, products.RemapKeysForProduct(req.QueryConfig.Fields))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if p != nil {
		out.Set("product", products.RemapProductResponse(p))
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := splitAdditional(body)
	rest.Set("product_id", req.Params["id"])
	if _, err := workflows.CreateProductVariantsWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateProductVariantsInput{ProductVariants: []*dal.Record{rest}, AdditionalData: additional}); err != nil {
		return err
	}
	return RespondProduct(req, res, req.Params["id"])
}

// SplitAdditional is splitAdditional for the nested routes.
var SplitAdditional = splitAdditional

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
