package carts

import "github.com/hitkey-io/medusago/packages/core/framework/src/zod"

// Validators.
var (
	// StoreAddGiftCardToCart mirrors StoreAddGiftCardToCart.
	StoreAddGiftCardToCart = zod.Object(zod.F("code", zod.String())).Strict()
	// StoreRemoveGiftCardFromCart mirrors StoreRemoveGiftCardFromCart.
	StoreRemoveGiftCardFromCart = zod.Object(zod.F("code", zod.String())).Strict()
	// StoreAddStoreCreditsToCart mirrors StoreAddStoreCreditsToCart.
	StoreAddStoreCreditsToCart = zod.Object(zod.F("amount", zod.Number().Optional()))
)
