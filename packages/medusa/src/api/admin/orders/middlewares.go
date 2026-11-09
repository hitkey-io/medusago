package orders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminOrderRoutesMiddlewares mirrors adminOrderRoutesMiddlewares (ported
// routes only).
var AdminOrderRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/orders", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersParams, ListTransformQueryConfig)}},
}
