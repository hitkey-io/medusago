// Package models mirrors the models of the loyalty module.
package models

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// GiftCard mirrors models/gift-card.ts.
var GiftCard = dml.Define(dml.DefineConfig{Name: "GiftCard", TableName: "loyalty_gift_card"}, dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "gcard"}).PrimaryKey()},
	{Name: "status", Property: dml.Enum("pending", "redeemed").Default("pending")},
	{Name: "value", Property: dml.BigNumber()},
	{Name: "code", Property: dml.Text().Searchable().Unique()},
	{Name: "currency_code", Property: dml.Text().Searchable()},
	{Name: "expires_at", Property: dml.DateTime().Nullable()},
	{Name: "reference_id", Property: dml.Text().Nullable()},
	{Name: "reference", Property: dml.Text().Nullable()},
	{Name: "line_item_id", Property: dml.Text().Nullable()},
	{Name: "note", Property: dml.Text().Nullable()},
	{Name: "metadata", Property: dml.JSON().Nullable()},
})

// All are the models of the module service (MedusaService({ GiftCard })).
var All = []*dml.Entity{GiftCard}
