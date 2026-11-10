package searchindexes

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminSearchIndexRoutesMiddlewares mirrors adminSearchIndexRoutesMiddlewares
// (authenticate is applied by the API loader; authorize is the Enterprise
// RBAC check, a no-op without the RBAC module).
var AdminSearchIndexRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/search-indexes/:id/reindex", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformBody(httputils.Schema(AdminReindexSearchIndex)),
	}},
}
