package giftcards

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminGiftCardMiddlewares (the RBAC authorize
// middlewares are not ported).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/gift-cards", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetGiftCardsParams, ListGiftCardsTransformQueryConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/gift-cards/:id", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetGiftCardsParams, RetrieveGiftCardTransformQueryConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/gift-cards/:id/orders", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetGiftCardsParams, RetrieveGiftCardOrdersTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/gift-cards", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateGiftCard)),
		httputils.ValidateAndTransformQuery(AdminGetGiftCardsParams, RetrieveGiftCardTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/gift-cards/:id", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminUpdateGiftCard)),
		httputils.ValidateAndTransformQuery(AdminGetGiftCardsParams, RetrieveGiftCardTransformQueryConfig))},
}
