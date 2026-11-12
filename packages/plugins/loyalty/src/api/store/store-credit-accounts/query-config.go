// Package storecreditaccounts mirrors api/store/store-credit-accounts of
// @medusajs/loyalty-plugin.
package storecreditaccounts

import (
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
)

// StoreCreditAccountFields mirrors storeCreditAccountFields.
var StoreCreditAccountFields = []string{"id", "currency_code", "customer_id", "*customer", "credits", "debits", "balance", "updated_at", "created_at", "metadata"}

// StoreCreditAccountScalarFields mirrors storeCreditAccountScalarFields.
var StoreCreditAccountScalarFields = []string{"id", "code", "currency_code", "customer_id", "credits", "debits", "balance", "updated_at", "created_at", "metadata"}

// AccountTransactionScalarFields mirrors accountTransactionScalarFields.
var AccountTransactionScalarFields = []string{"id", "amount", "type", "reference", "reference_id", "note", "metadata", "updated_at", "created_at"}

func prefixed(prefix string, fields []string) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = prefix + "." + f
	}
	return out
}

// StoreCreditAccountAllowedFields mirrors storeCreditAccountAllowedFields.
var StoreCreditAccountAllowedFields = storeutils.BuildAllowedFields(
	StoreCreditAccountFields,
	StoreCreditAccountScalarFields,
	[]string{"transactions"},
	prefixed("transactions", AccountTransactionScalarFields),
)

// Query configs.
var (
	RetrieveStoreCreditAccountTransformQueryConfig = httputils.QueryConfig{Defaults: StoreCreditAccountFields, Allowed: StoreCreditAccountAllowedFields}
	ListStoreCreditAccountsTransformQueryConfig    = httputils.QueryConfig{Defaults: StoreCreditAccountFields, Allowed: StoreCreditAccountAllowedFields, IsList: true}
)
