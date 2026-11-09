package productvariants

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// ListProductVariantQueryConfig mirrors listProductVariantQueryConfig.
var ListProductVariantQueryConfig = httputils.QueryConfig{Defaults: DefaultAdminProductVariantFields, DefaultLimit: 50, IsList: true}

// Middlewares mirrors adminProductVariantRoutesMiddlewares (authorize
// entries are Enterprise and skipped).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/product-variants", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetProductVariantsParams, ListProductVariantQueryConfig)}},
}
