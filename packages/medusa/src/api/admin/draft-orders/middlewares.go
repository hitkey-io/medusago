package draftorders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminDraftOrderRoutesMiddlewares mirrors adminDraftOrderRoutesMiddlewares
// (ported routes only; authorize is RBAC).
var AdminDraftOrderRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/draft-orders", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetDraftOrdersParams, ListTransformQueryConfig)}},
	{Methods: []string{"GET"}, Matcher: "/admin/draft-orders/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetDraftOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateDraftOrder)), httputils.ValidateAndTransformQuery(AdminGetDraftOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateDraftOrder)), httputils.ValidateAndTransformQuery(AdminGetDraftOrderParams, RetrieveTransformQueryConfig)}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/convert-to-order", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformQuery(AdminGetDraftOrderParams, RetrieveTransformQueryConfig)}},
}
