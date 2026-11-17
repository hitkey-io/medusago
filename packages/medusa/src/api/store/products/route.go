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

// translationListCacheTag mirrors TRANSLATION_LIST_CACHE_TAG.
const translationListCacheTag = "Translation:list:*"

// ListCacheOptions is the cache option of the store product and variant
// listings: { enable: true, tags: [TRANSLATION_LIST_CACHE_TAG],
// computeAutomaticTags: true }.
func ListCacheOptions() *caching.CacheOptions {
	c := caching.WithTags(true, translationListCacheTag)
	c.Enable = caching.Enabled().Enable
	return c
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	fields, withInventory := productmw.PrepareInventoryQuantityFields(req.QueryConfig.Fields, "variants")
	req.QueryConfig.Fields = fields
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{
		Entity:     "product",
		Fields:     req.QueryConfig.Fields,
		Filters:    req.FilterableFields,
		Pagination: req.QueryConfig.Pagination,
		Context:    PricingQueryContext(req),
	}, joiner.Options{Cache: ListCacheOptions(), Locale: req.Locale})
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
	// TODO upstream: remove estimate_count in the next minor release.
	if indexEngineEnabled(req) {
		out.Set("estimate_count", get(r.Metadata, "count"))
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
