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
		httputils.ApplyCrossModuleFilter(httputils.CrossModuleFilterOptions{FilterableField: "sales_channel_id", RelationPath: "sales_channels.id"}),
		httputils.ApplyCrossModuleFilter(httputils.CrossModuleFilterOptions{FilterableField: "price_list_id", RelationPath: "variants.prices.price_list_id"}),
	}},
	{Methods: []string{"POST"}, Matcher: "/admin/products", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateProduct)),
		httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig),
	}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/batch", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchProductsBody)), httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/import", Middlewares: []medusahttp.MiddlewareFunction{medusahttp.MulterSingle("file")}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/imports", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminImportProducts))}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/export", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductsParams, ListProductQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/products/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateProduct)), httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/products/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/products/:id/variants", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(productvariants.AdminGetProductVariantsParams, ListVariantConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id/variants", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateProductVariant)), httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id/variants/batch", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminBatchProductVariantsBody)), httputils.ValidateAndTransformQuery(AdminGetProductVariantParams, RetrieveVariantConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/products/:id/options", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductOptionsParams, ListOptionConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id/options/batch", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminLinkProductOptions)), httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/products/:id/variants/:variant_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductVariantParams, RetrieveVariantConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/products/:id/variants/:variant_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateProductVariant)), httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/products/:id/variants/:variant_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductParams, RetrieveProductQueryConfig)}},
}
