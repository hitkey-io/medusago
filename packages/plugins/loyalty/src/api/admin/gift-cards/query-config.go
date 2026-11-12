// Package giftcards mirrors api/admin/gift-cards of
// @medusajs/loyalty-plugin.
package giftcards

import httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"

// GiftCardFields mirrors giftCardFields.
var GiftCardFields = []string{
	"id", "status", "code", "value", "currency_code", "line_item_id", "reference_id", "reference",
	"expires_at", "note", "updated_at", "created_at", "metadata",
}

// GiftCardOrderFields mirrors giftCardOrderFields.
var GiftCardOrderFields = []string{"id", "display_id", "version", "status"}

// Query configs.
var (
	RetrieveGiftCardTransformQueryConfig       = httputils.QueryConfig{Defaults: GiftCardFields}
	ListGiftCardsTransformQueryConfig          = httputils.QueryConfig{Defaults: GiftCardFields, IsList: true}
	RetrieveGiftCardOrdersTransformQueryConfig = httputils.QueryConfig{Defaults: GiftCardOrderFields}
)
