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
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/edit/items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminAddDraftOrderItems))}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/edit/items/item/:item_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateDraftOrderItem))}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/edit/items/:action_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateDraftOrderActionItem))}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/edit/promotions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminAddDraftOrderPromotions))}},
	{Methods: []string{"DELETE"}, Matcher: "/admin/draft-orders/:id/edit/promotions", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminRemoveDraftOrderPromotions))}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/edit/shipping-methods", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminAddDraftOrderShippingMethod))}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/edit/shipping-methods/method/:method_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateDraftOrderShippingMethod))}},
	{Methods: []string{"POST"}, Matcher: "/admin/draft-orders/:id/edit/shipping-methods/:action_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateDraftOrderActionShippingMethod))}},
}
