package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	productvariants "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/product-variants"
)

// AdminProductRoutesMiddlewares mirrors adminProductRoutesMiddlewares
// (ported routes only; authorize is RBAC).
var AdminProductRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/products", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(AdminGetProductsParams, ListProductQueryConfig),
		httputils.MaybeApplyLinkFilter(httputils.LinkFilterOptions{EntryPoint: "product_sales_channel", ResourceID: "product_id", FilterableField: "sales_channel_id"}),
		MaybeApplyPriceListsFilter(),
	}},
	{Methods: []string{"POST"}, Matcher: "/admin/products", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateProduct)),
		httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig),
	}},
	{Methods: []string{"GET"}, Matcher: "/admin/products/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateProduct)), httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/products/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/products/:id/variants", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(productvariants.AdminGetProductVariantsParams, ListVariantConfig)}},
}
