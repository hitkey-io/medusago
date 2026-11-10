package claims

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

func q() medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(AdminGetOrdersOrderParams, RetrieveTransformQueryConfig)
}

// AdminClaimRoutesMiddlewares mirrors adminClaimRoutesMiddlewares (ported
// routes only; authorize is RBAC).
var AdminClaimRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/claims", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetOrdersParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/claims/:id", Middlewares: []medusahttp.MiddlewareFunction{q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/claims", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderClaimsReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/claims/:id/claim-items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostClaimItemsReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/claims/:id/inbound/items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostClaimsRequestReturnItemsReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/claims/:id/inbound/shipping-method", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostClaimsShippingReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/claims/:id/outbound/items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostClaimsAddItemsReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/claims/:id/outbound/shipping-method", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostClaimsShippingReqSchema)), q()}},
	{Methods: []string{"POST"}, Matcher: "/admin/claims/:id/request", Middlewares: []medusahttp.MiddlewareFunction{q()}},
}
