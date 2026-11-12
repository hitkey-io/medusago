// Package carts mirrors api/store/carts of @medusajs/loyalty-plugin.
package carts

import (
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	storecarts "github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
	giftcards "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/gift-cards"
)

// RetrieveTransformQueryConfig mirrors retrieveTransformQueryConfig: the
// store cart fields with the gift cards.
var RetrieveTransformQueryConfig = httputils.QueryConfig{
	Defaults: append(append([]string{}, storecarts.DefaultStoreCartFields...), "*gift_cards"),
	Allowed:  append(append([]string{}, storecarts.RetrieveConfig.Allowed...), giftcards.GiftCardRelationAllowedFields...),
}
