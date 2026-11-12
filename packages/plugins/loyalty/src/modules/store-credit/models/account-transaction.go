package models

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// AccountTransaction mirrors models/account-transaction.ts.
var AccountTransaction = dml.Define(dml.DefineConfig{Name: "AccountTransaction", TableName: "store_credit_account_transaction"}, dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "sc_trx"}).PrimaryKey()},
	{Name: "amount", Property: dml.BigNumber()},
	{Name: "type", Property: dml.Enum("credit", "debit")},
	{Name: "reference", Property: dml.Text()},
	{Name: "reference_id", Property: dml.Text()},
	{Name: "note", Property: dml.Text().Nullable()},
	{Name: "account", Property: dml.BelongsTo("StoreCreditAccount", dml.RelationshipOptions{MappedBy: "transactions"})},
	{Name: "metadata", Property: dml.JSON().Nullable()},
})

// All are the models of the module service, in the order of
// models/index.ts (AccountTransaction, StoreCreditAccount).
var All = []*dml.Entity{AccountTransaction, StoreCreditAccount}
