package carts

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	storecarts "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

func query() medusahttp.MiddlewareFunction {
	return httputils.ValidateAndTransformQuery(storecarts.StoreGetCartsCart, RetrieveTransformQueryConfig)
}

// Middlewares mirrors storeCartMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id/gift-cards", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(StoreAddGiftCardToCart)), query())},
	{Methods: []string{"DELETE"}, Matcher: "/store/carts/:id/gift-cards", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(StoreRemoveGiftCardFromCart)), query())},
	{Methods: []string{"POST"}, Matcher: "/store/carts/:id/store-credits", Middlewares: mw(
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}),
		httputils.ValidateAndTransformBody(httputils.Schema(StoreAddStoreCreditsToCart)), query())},
}
