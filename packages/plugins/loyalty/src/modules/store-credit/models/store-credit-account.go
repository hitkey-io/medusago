// Package models mirrors the models of the store credit module.
package models

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// StoreCreditAccount mirrors models/store-credit-account.ts.
var StoreCreditAccount = dml.Define(dml.DefineConfig{Name: "StoreCreditAccount", TableName: "store_credit_account"}, dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "sc_acc"}).PrimaryKey()},
	{Name: "code", Property: dml.Text().Nullable()},
	{Name: "currency_code", Property: dml.Text().Searchable()},
	{Name: "customer_id", Property: dml.Text().Nullable()},
	{Name: "metadata", Property: dml.JSON().Nullable()},
	{Name: "transactions", Property: dml.HasMany("AccountTransaction", dml.RelationshipOptions{MappedBy: "account"})},
}).WithIndexes([]dml.Index{{
	Name: "IDX_customer_id_currency_code", On: []string{"customer_id", "currency_code"}, Unique: true, Where: "customer_id IS NOT NULL",
}})
