package orders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminOrderRoutesMiddlewares mirrors adminOrderRoutesMiddlewares (ported
// routes only).
var AdminOrderRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/orders", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/orders/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/orders/:id/line-items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderItemsParams, ListOrderItemsQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/orders/:id/changes", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminOrderChangesParams, RetrieveOrderChangesTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/orders/:id/preview", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminOrderCreateFulfillment)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments/:fulfillment_id/cancel", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminOrderCancelFulfillment)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments/:fulfillment_id/shipments", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminOrderCreateShipment)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments/:fulfillment_id/mark-as-delivered", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig), httputils.ValidateAndTransformBody(httputils.Schema(AdminMarkOrderFulfillmentAsDelivered))}},
}
