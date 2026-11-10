package returns

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminReturnRoutesMiddlewares mirrors adminReturnRoutesMiddlewares (ported
// routes only; authorize is RBAC).
var AdminReturnRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/returns", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/returns/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetReturnParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/returns", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostReturnsReqSchema)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/returns/:id/request-items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostReturnsRequestItemsReqSchema)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/returns/:id/shipping-method", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostReturnsShippingReqSchema)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/returns/:id/request", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostReturnsConfirmRequestReqSchema)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/returns/:id/receive", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostReceiveReturnsReqSchema)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/returns/:id/receive/confirm", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostReturnsConfirmRequestReqSchema)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/returns/:id/receive-items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostReceiveReturnItemsReqSchema)), httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)}},
}
