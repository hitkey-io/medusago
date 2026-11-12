package api

import medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"

// The store credit account routes: admin/store-credit-accounts and
// store/store-credit-accounts, with their middlewares.
var (
	storeCreditRoutes                  []medusahttp.RouteFile
	adminStoreCreditAccountMiddlewares []medusahttp.MiddlewareRoute
	storeStoreCreditAccountMiddlewares []medusahttp.MiddlewareRoute
)
