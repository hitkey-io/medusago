package regions

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminRegionRoutesMiddlewares mirrors adminRegionRoutesMiddlewares
// (authorize entries are Enterprise and skipped).
var AdminRegionRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/regions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetRegionsParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/regions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetRegionParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/regions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateRegion)), httputils.ValidateAndTransformQuery(AdminGetRegionParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/regions/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateRegion)), httputils.ValidateAndTransformQuery(AdminGetRegionParams, RetrieveTransformQueryConfig)}},
}
