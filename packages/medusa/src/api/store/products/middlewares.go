package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	productmw "github.com/hitkey-io/medusago/packages/medusa/src/api/utils/middlewares/products"
)

var allowed = storeutils.BuildAllowedFields(DefaultStoreProductFields, AllowedStoreProductExtraFields)

// Query configs.
var (
	RetrieveProductQueryConfig = httputils.QueryConfig{Defaults: DefaultStoreProductFields, Allowed: allowed, StoreRelationsLimit: 4}
	ListProductQueryConfig     = httputils.QueryConfig{Defaults: DefaultStoreProductFields, Allowed: allowed, StoreRelationsLimit: 4, DefaultLimit: 50, IsList: true}
)

var productSalesChannelLink = httputils.LinkFilterOptions{EntryPoint: "product_sales_channel", ResourceID: "product_id", FilterableField: "sales_channel_id"}

// applyMaybeLinkFilterIfNecessary mirrors applyMaybeLinkFilterIfNecessary
// (index engine disabled).
func applyMaybeLinkFilterIfNecessary(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	p := dal.NewRecord()
	p.Set("skip", float64(0))
	p.Set("take", float64(1))
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "sales_channels", Fields: []string{"id"}, Pagination: p}, joiner.Options{})
	if err != nil {
		return err
	}
	count := float64(0)
	if r.Metadata != nil {
		if c, ok := r.Metadata.Get("count"); ok {
			count, _ = c.(float64)
		}
	}
	if !(count > 1) {
		req.FilterableFields.Delete("sales_channel_id")
		next(nil)
		return nil
	}
	return httputils.MaybeApplyLinkFilter(productSalesChannelLink)(req, res, next)
}

func authenticateCustomer() medusahttp.MiddlewareFunction {
	return middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{AllowUnauthenticated: true})
}

var published = middlewares.DefaultFilter{Key: "status", Value: "published"}

// Middlewares mirrors storeProductRoutesMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/store/products", Middlewares: []medusahttp.MiddlewareFunction{
		authenticateCustomer(),
		httputils.ValidateAndTransformQuery(StoreGetProductsParams, ListProductQueryConfig),
		productmw.FilterByValidSalesChannels(),
		applyMaybeLinkFilterIfNecessary,
		middlewares.ApplyDefaultFilters(published, middlewares.DefaultFilter{Key: "categories", Fn: func(filters *dal.Record, fields []string) any {
			ids, _ := filters.Get("category_id")
			filters.Delete("category_id")
			if !present(ids) {
				return nil
			}
			r := dal.NewRecord()
			r.Set("id", ids)
			r.Set("is_internal", false)
			r.Set("is_active", true)
			return r
		}}),
		productmw.NormalizeDataForContext(),
		productmw.SetPricingContext(),
		productmw.SetTaxContext(),
		middlewares.ClearFiltersByKey("region_id", "country_code", "province", "cart_id"),
	}},
	{Methods: []string{"GET"}, Matcher: "/store/products/:id", Middlewares: []medusahttp.MiddlewareFunction{
		authenticateCustomer(),
		httputils.ValidateAndTransformQuery(StoreGetProductsParams, RetrieveProductQueryConfig),
		middlewares.ApplyParamsAsFilters(map[string]string{"id": "id"}),
		productmw.FilterByValidSalesChannels(),
		httputils.MaybeApplyLinkFilter(productSalesChannelLink),
		middlewares.ApplyDefaultFilters(published),
		productmw.NormalizeDataForContext(),
		productmw.SetPricingContext(),
		productmw.SetTaxContext(),
		middlewares.ClearFiltersByKey("region_id", "country_code", "province", "cart_id"),
	}},
}

func present(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t.Len() > 0
	}
	return true
}

// PricingQueryContext builds { variants: { calculated_price:
// QueryContext(pricingContext) } } (empty without a pricing context).
func PricingQueryContext(req *medusahttp.MedusaRequest) *dal.Record {
	ctx := dal.NewRecord()
	if req.PricingContext != nil && req.PricingContext.Len() > 0 {
		v := dal.NewRecord()
		v.Set("calculated_price", query.QueryContext(req.PricingContext))
		ctx.Set("variants", v)
	}
	return ctx
}
