package orderedits

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// AdminOrderEditRoutesMiddlewares mirrors adminOrderEditRoutesMiddlewares
// (ported routes only; authorize is RBAC).
var AdminOrderEditRoutesMiddlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/admin/order-edits", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderEditsReqSchema))}},
	{Methods: []string{"POST"}, Matcher: "/admin/order-edits/:id/items", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderEditsAddItemsReqSchema))}},
	{Methods: []string{"POST"}, Matcher: "/admin/order-edits/:id/items/item/:item_id", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderEditsUpdateItemQuantityReqSchema))}},
	{Methods: []string{"POST"}, Matcher: "/admin/order-edits/:id/shipping-method", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderEditsShippingReqSchema))}},
	{Methods: []string{"POST"}, Matcher: "/admin/order-edits/:id/request", Middlewares: []medusahttp.MiddlewareFunction{httputils.ValidateAndTransformBody(httputils.Schema(AdminPostOrderEditsRequestReqSchema))}},
}
