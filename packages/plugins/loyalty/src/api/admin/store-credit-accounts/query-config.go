// Package storecreditaccounts mirrors api/admin/store-credit-accounts of
// @medusajs/loyalty-plugin.
package storecreditaccounts

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// StoreCreditAccountFields mirrors storeCreditAccountFields.
var StoreCreditAccountFields = []string{"id", "currency_code", "customer_id", "code", "*customer", "credits", "debits", "balance", "updated_at", "created_at", "metadata"}

// StoreCreditAccountTransactionsFields mirrors storeCreditAccountTransactionsFields.
var StoreCreditAccountTransactionsFields = []string{"id", "amount", "type", "reference", "reference_id", "note", "updated_at", "created_at", "metadata"}

// Query configs.
var (
	RetrieveStoreCreditAccountTransformQueryConfig             = httputils.QueryConfig{Defaults: StoreCreditAccountFields}
	ListStoreCreditAccountsTransformQueryConfig                = httputils.QueryConfig{Defaults: StoreCreditAccountFields, IsList: true}
	RetrieveStoreCreditAccountTransactionsTransformQueryConfig = httputils.QueryConfig{Defaults: StoreCreditAccountTransactionsFields}
	ListStoreCreditAccountTransactionsTransformQueryConfig     = httputils.QueryConfig{Defaults: StoreCreditAccountTransactionsFields, IsList: true}
)
