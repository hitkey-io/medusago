// Package id_ mirrors api/store/products/[id].
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/products"
	productmw "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/middlewares/products"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	fields, withInventory := productmw.PrepareInventoryQuantityFields(req.QueryConfig.Fields, "variants")
	req.QueryConfig.Fields = fields
	filters := dal.NewRecord()
	filters.Set("id", req.Params["id"])
	if req.FilterableFields != nil {
		for _, k := range req.FilterableFields.Keys() {
			v, _ := req.FilterableFields.Get(k)
			filters.Set(k, v)
		}
	}
	includesCategories := false
	hasInternal := false
	for _, f := range req.QueryConfig.Fields {
		if len(f) >= 10 && f[:10] == "categories" {
			includesCategories = true
		}
		if f == "categories.is_internal" {
			hasInternal = true
		}
	}
	if !hasInternal {
		req.QueryConfig.Fields = append(req.QueryConfig.Fields, "categories.is_internal")
	}
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{
		Entity:  "product",
		Filters: filters,
		Context: products.PricingQueryContext(req),
		Fields:  req.QueryConfig.Fields,
	}, joiner.Options{})
	if err != nil {
		return err
	}
	var product *dal.Record
	if len(r.Data) > 0 {
		product, _ = r.Data[0].(*dal.Record)
	}
	if product == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Product with id: %s was not found", req.Params["id"]))
	}
	if withInventory {
		variants, _ := product.Get("variants")
		list, _ := variants.([]any)
		if err := products.WrapInventoryQuantity(req, list); err != nil {
			return err
		}
	}
	if includesCategories {
		products.FilterOutInternalProductCategories([]*dal.Record{product})
	}
	if err := products.WrapProductsWithTaxPrices(req, []*dal.Record{product}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("product", product)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
