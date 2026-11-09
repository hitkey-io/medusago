package stores

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminStoreRoutesMiddlewares mirrors adminStoreRoutesMiddlewares (ported
// routes only).
var AdminStoreRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/stores", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetStoresParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/stores/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetStoreParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/stores/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateStore)), httputils.ValidateAndTransformQuery(AdminGetStoreParams, RetrieveTransformQueryConfig)}},
}
