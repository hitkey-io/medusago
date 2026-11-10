package exchanges

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

func q() medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)
}

// AdminExchangeRoutesMiddlewares mirrors adminExchangeRoutesMiddlewares (ported
// routes only; authorize is RBAC).
var AdminExchangeRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/exchanges", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/exchanges/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetExchangeParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderExchangesReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/inbound/items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesReturnRequestItemsReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/inbound/shipping-method", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesShippingReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/outbound/items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesAddItemsReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/outbound/shipping-method", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesShippingReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/request", Middlewares: []medusahttp.MiddlewareFunction{q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/inbound/items/:action_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesRequestItemsReturnActionReqSchema)), q()}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/exchanges/:id/inbound/items/:action_id", Middlewares: []medusahttp.MiddlewareFunction{q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/inbound/shipping-method/:action_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesShippingActionReqSchema)), q()}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/exchanges/:id/inbound/shipping-method/:action_id", Middlewares: []medusahttp.MiddlewareFunction{q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/outbound/items/:action_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesItemsActionReqSchema)), q()}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/exchanges/:id/outbound/items/:action_id", Middlewares: []medusahttp.MiddlewareFunction{q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/outbound/shipping-method/:action_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostExchangesShippingActionReqSchema)), q()}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/exchanges/:id/outbound/shipping-method/:action_id", Middlewares: []medusahttp.MiddlewareFunction{q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/exchanges/:id/cancel", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostCancelExchangeReqSchema)), q()}},
}
