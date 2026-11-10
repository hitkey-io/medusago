package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	featureflags "github.com/hitkey-io/medusago/packages/medusa/src/feature-flags"
)

// indexEngineEnabled mirrors FeatureFlag.isFeatureEnabled(IndexEngineFeatureFlag.key).
func indexEngineEnabled(req *medusahttp.MedusaRequest) bool {
	return featureflags.IsFeatureEnabled(req.Scope, featureflags.IndexEngine.Key)
}

func filterValue(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// child returns filters[key], creating it (`filters[key] ??= {}`).
func child(r *dal.Record, key string) *dal.Record {
	if v, ok := filterValue(r, key).(*dal.Record); ok {
		return v
	}
	c := dal.NewRecord()
	r.Set(key, c)
	return c
}

// useIndexEngine mirrors the GET guard: the regular list when no filter is
// given or when filtering on what the index does not hold.
func useIndexEngine(req *medusahttp.MedusaRequest) bool {
	ff := req.FilterableFields
	if ff == nil || ff.Len() == 0 {
		return false
	}
	for _, k := range []string{"tags", "categories", "option_value_id", "options"} {
		if featureflags.IsPresent(filterValue(ff, k)) {
			return false
		}
	}
	return true
}

// getProductsWithIndexEngine mirrors getProductsWithIndexEngine.
func getProductsWithIndexEngine(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := req.FilterableFields
	if ids := filterValue(filters, "sales_channel_id"); featureflags.IsPresent(ids) {
		child(filters, "sales_channels").Set("id", ids)
		filters.Delete("sales_channel_id")
	}
	if ids := filterValue(filters, "price_list_id"); featureflags.IsPresent(ids) {
		child(child(filters, "variants"), "prices").Set("price_list_id", ids)
		filters.Delete("price_list_id")
	}
	// TODO upstream: search by relations like query.graph
	if featureflags.IsPresent(filterValue(filters, "q")) {
		child(filters, "variants")
	}
	withDeleted := req.QueryConfig.WithDeleted != nil && *req.QueryConfig.WithDeleted
	fields := req.QueryConfig.Fields
	if fields == nil {
		fields = []string{}
	}
	r, err := httputils.QueryOf(req.Scope).Index(req.Ctx(), query.GraphInput{
		Entity:      "product",
		Fields:      fields,
		Filters:     filters,
		Pagination:  req.QueryConfig.Pagination,
		WithDeleted: withDeleted,
	}, nil, joiner.Options{})
	if err != nil {
		return err
	}
	products := make([]any, len(r.Data))
	for i, p := range r.Data {
		products[i] = RemapProductResponse(p)
	}
	out := dal.NewRecord()
	out.Set("products", products)
	out.Set("count", filterValue(r.Metadata, "estimate_count"))
	out.Set("estimate_count", filterValue(r.Metadata, "estimate_count"))
	out.Set("offset", filterValue(r.Metadata, "skip"))
	out.Set("limit", filterValue(r.Metadata, "take"))
	return res.JSON(out)
}

// skipWithIndexEngine runs the middleware only when the index engine is
// off (the sales channel link filter of GET /admin/products).
func skipWithIndexEngine(mw medusahttp.MiddlewareFunction) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		if indexEngineEnabled(req) {
			next(nil)
			return nil
		}
		return mw(req, res, next)
	}
}
