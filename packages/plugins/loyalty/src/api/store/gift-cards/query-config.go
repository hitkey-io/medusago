// Package giftcards mirrors api/store/gift-cards of
// @medusajs/loyalty-plugin.
package giftcards

import (
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
	storecreditaccounts "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/store-credit-accounts"
)

// GiftCardFields mirrors giftCardFields.
var GiftCardFields = []string{"id", "status", "code", "value", "currency_code", "expires_at", "note", "updated_at", "created_at"}

// GiftCardScalarFields mirrors giftCardScalarFields.
var GiftCardScalarFields = append(append([]string{}, GiftCardFields...), "reference", "reference_id", "line_item_id", "metadata")

var lineItemScalarFields = []string{
	"id", "title", "subtitle", "thumbnail", "variant_id", "product_id", "product_title", "product_description",
	"product_subtitle", "product_type_id", "product_type", "product_collection", "product_handle", "variant_sku",
	"variant_barcode", "variant_title", "variant_option_values", "requires_shipping", "is_discountable",
	"is_tax_inclusive", "unit_price", "quantity", "created_at", "updated_at",
}

func prefixed(relation string, fields []string) []string {
	out := []string{relation}
	for _, f := range fields {
		out = append(out, relation+"."+f)
	}
	return out
}

// GiftCardAllowedFields mirrors giftCardAllowedFields.
var GiftCardAllowedFields = storeutils.BuildAllowedFields(
	GiftCardScalarFields,
	prefixed("store_credit_account", storecreditaccounts.StoreCreditAccountScalarFields),
	prefixed("line_item", lineItemScalarFields),
)

// GiftCardRelationAllowedFields mirrors giftCardRelationAllowedFields.
var GiftCardRelationAllowedFields = prefixed("gift_cards", GiftCardAllowedFields)

// Query configs.
var (
	RetrieveGiftCardTransformQueryConfig = httputils.QueryConfig{Defaults: GiftCardFields, Allowed: GiftCardAllowedFields}
	ListGiftCardsTransformQueryConfig    = httputils.QueryConfig{Defaults: GiftCardFields, Allowed: GiftCardAllowedFields, IsList: true}
)
