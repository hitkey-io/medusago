package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	productmw "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/middlewares/products"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
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

// GET mirrors GET: with the index engine, query.index unless filtering on
// what the index does not hold (tags, categories, options).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if indexEngineEnabled(req) {
		for _, k := range []string{"tags", "categories", "option_value_id", "options"} {
			if featureflags.IsPresent(get(req.FilterableFields, k)) {
				return getProducts(req, res, false)
			}
		}
		return getProducts(req, res, true)
	}
	return getProducts(req, res, false)
}

// getProducts mirrors getProducts, and getProductsWithIndexEngine when
// withIndex is set.
func getProducts(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, withIndex bool) error {
	fields, withInventory := productmw.PrepareInventoryQuantityFields(req.QueryConfig.Fields, "variants")
	req.QueryConfig.Fields = fields
	var r *query.GraphResult
	var err error
	if withIndex {
		filters := req.FilterableFields
		if ids := get(filters, "sales_channel_id"); featureflags.IsPresent(ids) {
			sc, ok := get(filters, "sales_channels").(*dal.Record)
			if !ok {
				sc = dal.NewRecord()
				filters.Set("sales_channels", sc)
			}
			sc.Set("id", ids)
			filters.Delete("sales_channel_id")
		}
		if featureflags.IsPresent(get(filters, "q")) {
			if _, ok := get(filters, "variants").(*dal.Record); !ok {
				filters.Set("variants", dal.NewRecord())
			}
		}
		r, err = httputils.QueryOf(req.Scope).Index(req.Ctx(), query.GraphInput{
			Entity:     "product",
			Fields:     req.QueryConfig.Fields,
			Filters:    filters,
			Pagination: req.QueryConfig.Pagination,
			Context:    PricingQueryContext(req),
		}, nil, joiner.Options{Cache: caching.Enabled(), Locale: req.Locale})
	} else {
		r, err = httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{
			Entity:     "product",
			Fields:     req.QueryConfig.Fields,
			Filters:    req.FilterableFields,
			Pagination: req.QueryConfig.Pagination,
			Context:    PricingQueryContext(req),
		}, joiner.Options{Cache: caching.Enabled(), Locale: req.Locale})
	}
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
	if withIndex {
		out.Set("count", get(r.Metadata, "estimate_count"))
		out.Set("estimate_count", get(r.Metadata, "estimate_count"))
	} else {
		out.Set("count", get(r.Metadata, "count"))
	}
	out.Set("offset", get(r.Metadata, "skip"))
	out.Set("limit", get(r.Metadata, "take"))
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}

// indexEngineEnabled mirrors FeatureFlag.isFeatureEnabled(IndexEngineFeatureFlag.key).
func indexEngineEnabled(req *medusahttp.MedusaRequest) bool {
	return featureflags.IsFeatureEnabled(req.Scope, featureflags.IndexEngine.Key)
}
