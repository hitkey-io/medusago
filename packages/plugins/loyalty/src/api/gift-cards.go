package api

import medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"

// The gift card routes: admin/gift-cards, store/gift-cards and store/carts
// (both cart routes), with their middlewares.
var (
	giftCardRoutes            []medusahttp.RouteFile
	allowGiftCardRelation     []medusahttp.MiddlewareRoute
	adminGiftCardMiddlewares  []medusahttp.MiddlewareRoute
	storeGiftCardsMiddlewares []medusahttp.MiddlewareRoute
	storeCartMiddlewares      []medusahttp.MiddlewareRoute
)
