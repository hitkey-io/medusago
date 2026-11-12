package storecreditaccounts

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

func mw(fns ...medusahttp.MiddlewareFunction) []medusahttp.MiddlewareFunction { return fns }

// Middlewares mirrors adminStoreCreditAccountMiddlewares (the RBAC
// authorize middlewares are not ported).
var Middlewares = []medusahttp.MiddlewareRoute{
	{Methods: []string{"GET"}, Matcher: "/admin/store-credit-accounts", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetStoreCreditAccountsParams, ListStoreCreditAccountsTransformQueryConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/store-credit-accounts/:id", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetStoreCreditAccountsParams, RetrieveStoreCreditAccountTransformQueryConfig))},
	{Methods: []string{"GET"}, Matcher: "/admin/store-credit-accounts/:id/transactions", Middlewares: mw(
		httputils.ValidateAndTransformQuery(AdminGetStoreCreditAccountTransactionsParams, ListStoreCreditAccountTransactionsTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/store-credit-accounts/:id/credit", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreditStoreCreditAccountParams)),
		httputils.ValidateAndTransformQuery(AdminGetStoreCreditAccountsParams, RetrieveStoreCreditAccountTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/store-credit-accounts/:id/debit", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminDebitStoreCreditAccountParams)),
		httputils.ValidateAndTransformQuery(AdminGetStoreCreditAccountsParams, RetrieveStoreCreditAccountTransformQueryConfig))},
	{Methods: []string{"POST"}, Matcher: "/admin/store-credit-accounts", Middlewares: mw(
		httputils.ValidateAndTransformBody(httputils.Schema(AdminCreateStoreCreditAccount)),
		httputils.ValidateAndTransformQuery(AdminGetStoreCreditAccountsParams, RetrieveStoreCreditAccountTransformQueryConfig))},
}
