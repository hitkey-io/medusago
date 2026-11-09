package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productmw "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/middlewares/products"
)

// WrapInventoryQuantity mirrors wrapVariantsWithInventoryQuantityForSalesChannel(req, variants).
func WrapInventoryQuantity(req *medusahttp.MedusaRequest, variants []any) error {
	var keyIDs []any
	if v := get(req.PublishableKeyContext, "sales_channel_ids"); v != nil {
		keyIDs, _ = v.([]any)
	}
	reqIDs, err := productmw.TransformAndValidateSalesChannelIDs(req)
	if err != nil {
		return err
	}
	return productmw.WrapVariantsWithInventoryQuantityForSalesChannel(req.Ctx(), httputils.QueryOf(req.Scope), variants, reqIDs, keyIDs)
}

// GET mirrors GET (getProducts; the index engine is disabled).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	fields, withInventory := productmw.PrepareInventoryQuantityFields(req.QueryConfig.Fields, "variants")
	req.QueryConfig.Fields = fields
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{
		Entity:     "product",
		Fields:     req.QueryConfig.Fields,
		Filters:    req.FilterableFields,
		Pagination: req.QueryConfig.Pagination,
		Context:    PricingQueryContext(req),
	}, joiner.Options{})
	if err != nil {
		return err
	}
	products := records(r.Data)
	if withInventory {
		var variants []any
		for _, p := range products {
			if vs, ok := get(p, "variants").([]any); ok {
				variants = append(variants, vs...)
			}
		}
		if err := WrapInventoryQuantity(req, variants); err != nil {
			return err
		}
	}
	if err := WrapProductsWithTaxPrices(req, products); err != nil {
		return err
	}
	data := r.Data
	if data == nil {
		data = []any{}
	}
	out := dal.NewRecord()
	out.Set("products", data)
	out.Set("count", get(r.Metadata, "count"))
	out.Set("offset", get(r.Metadata, "skip"))
	out.Set("limit", get(r.Metadata, "take"))
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
