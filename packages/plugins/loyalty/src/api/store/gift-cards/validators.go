package giftcards

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/utils"
)

// Validators.
var (
	// StoreGetGiftCardParams mirrors StoreGetGiftCardParams.
	StoreGetGiftCardParams = utils.CreateSelectParams()
	// StoreRedeemGiftCard mirrors StoreRedeemGiftCard.
	StoreRedeemGiftCard = zod.Object().Strict()
)
