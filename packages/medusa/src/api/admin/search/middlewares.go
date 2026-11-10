package search

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminSearchRoutesMiddlewares mirrors adminSearchRoutesMiddlewares.
var AdminSearchRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/search", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformQuery(AdminGetSearchParams, httputils.QueryConfig{IsList: true, DefaultLimit: 20}),
	}},
}
