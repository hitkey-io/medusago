package fulfillments

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

func q() medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(AdminFulfillmentParams, RetrieveTransformQueryConfig)
}

// AdminFulfillmentsRoutesMiddlewares mirrors adminFulfillmentsRoutesMiddlewares
// (authorize is RBAC).
var AdminFulfillmentsRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/fulfillments/:id/cancel", Middlewares: []medusahttp.MiddlewareFunction{q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/fulfillments", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateFulfillment)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/fulfillments/:id/shipment", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateShipment)), q()}},
}
