package orderchanges

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminOrderChangesMiddlewares mirrors adminOrderChangesMiddlewares
// (authorize is RBAC).
var AdminOrderChangesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/order-changes/:id", Middlewares: []medusahttp.MiddlewareFunction{
		httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderChangesReqSchema)),
		httputils.ValidateAndTransformQuery(AdminOrderChangeParams, RetrieveTransformQueryConfig),
	}},
}
