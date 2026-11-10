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
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateOrder)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/orders/:id/shipping-options", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrderShippingOptionList, ListShippingOptionsQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/payment-sessions/authorize", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminAuthorizeOrderPaymentSession)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/credit-lines", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateOrderCreditLines)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/archive", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/cancel", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/complete", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCompleteOrder)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminOrderCreateFulfillment)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments/:fulfillment_id/cancel", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminOrderCancelFulfillment)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments/:fulfillment_id/shipments", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminOrderCreateShipment)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/fulfillments/:fulfillment_id/mark-as-delivered", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig), httputils.ValidateAndTransformBody(httputils.Schema(AdminMarkOrderFulfillmentAsDelivered))}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/transfer", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminTransferOrder)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/transfer/guest", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminTransferOrderToGuest)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/orders/:id/transfer/cancel", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
}
