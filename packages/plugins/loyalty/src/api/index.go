// Package api mirrors the API routes of @medusajs/loyalty-plugin (src/api).
package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

// Routes are the route files of src/api.
func Routes() []medusahttp.RouteFile {
	out := append([]medusahttp.RouteFile{}, giftCardRoutes...)
	return append(out, storeCreditRoutes...)
}

// Middlewares mirrors middlewares.ts (defineMiddlewares, in its order).
func Middlewares() []medusahttp.MiddlewareRoute {
	var out []medusahttp.MiddlewareRoute
	for _, part := range [][]medusahttp.MiddlewareRoute{
		allowGiftCardRelation,
		adminGiftCardMiddlewares,
		storeGiftCardsMiddlewares,
		adminStoreCreditAccountMiddlewares,
		storeStoreCreditAccountMiddlewares,
		storeCartMiddlewares,
	} {
		out = append(out, part...)
	}
	return out
}
