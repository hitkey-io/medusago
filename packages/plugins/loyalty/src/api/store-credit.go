package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	adminaccounts "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts"
	adminaccount "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts/id_"
	adminaccountcredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts/id_/credit"
	adminaccountdebit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts/id_/debit"
	adminaccounttransactions "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/store-credit-accounts/id_/transactions"
	storeaccounts "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/store-credit-accounts"
	storeaccountclaim "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/store-credit-accounts/claim"
	storeaccount "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/store-credit-accounts/id_"
)

// The store credit account routes: admin/store-credit-accounts and
// store/store-credit-accounts, with their middlewares.
var (
	storeCreditRoutes = []medusahttp.RouteFile{
		{Path: "/admin/store-credit-accounts", Route: adminaccounts.Route},
		{Path: "/admin/store-credit-accounts/:id", Route: adminaccount.Route},
		{Path: "/admin/store-credit-accounts/:id/credit", Route: adminaccountcredit.Route},
		{Path: "/admin/store-credit-accounts/:id/debit", Route: adminaccountdebit.Route},
		{Path: "/admin/store-credit-accounts/:id/transactions", Route: adminaccounttransactions.Route},
		{Path: "/store/store-credit-accounts", Route: storeaccounts.Route},
		{Path: "/store/store-credit-accounts/:id", Route: storeaccount.Route},
		{Path: "/store/store-credit-accounts/claim", Route: storeaccountclaim.Route},
	}
	adminStoreCreditAccountMiddlewares = adminaccounts.Middlewares
	storeStoreCreditAccountMiddlewares = storeaccounts.Middlewares
)
