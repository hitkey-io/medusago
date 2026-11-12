package storecreditaccounts

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors storeStoreCreditAccountMiddlewares.
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"ALL"}, Matcher: "/store/store-credit-accounts*", Middlewares: mw(
		middlewares.Authenticate([]string{"customer"}, []string{"session", "bearer"}, medusahttp.AuthenticateOptions{}))},
	{Methods: []string{"GET"}, Matcher: "/store/store-credit-accounts", Middlewares: mw(
		httputils.ValidateAndTransformQuery(StoreGetStoreCreditAccountsParams, ListStoreCreditAccountsTransformQueryConfig))},
	{Methods: []string{"GET"}, Matcher: "/store/store-credit-accounts/:id", Middlewares: mw(
		httputils.ValidateAndTransformQuery(StoreGetStoreCreditAccountParams, RetrieveStoreCreditAccountTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/store/store-credit-accounts/claim", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(StoreClaimStoreCreditAccountParams)),
		httputils.ValidateAndTransformQuery(StoreGetStoreCreditAccountParams, RetrieveStoreCreditAccountTransformQueryConfig))},
}
