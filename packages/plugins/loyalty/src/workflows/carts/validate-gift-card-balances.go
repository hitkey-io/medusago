package carts

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ValidateGiftCardBalancesStepInput mirrors ValidateGiftCardBalancesStepInput.
type ValidateGiftCardBalancesStepInput struct {
	GiftCards []*dal.Record `json:"giftCards"`
	// GiftCardsBalanceMap maps gift card codes to their account stats.
	GiftCardsBalanceMap *dal.Record `json:"giftCardsBalanceMap"`
}

// ValidateGiftCardBalancesStep mirrors validateGiftCardBalancesStep
// ("validate-gift-card-balances"): every gift card must have a positive
// balance.
var ValidateGiftCardBalancesStep = composer.CreateStep("validate-gift-card-balances",
	func(ctx composer.StepExecutionContext, in ValidateGiftCardBalancesStepInput) (composer.StepResponse[any], error) {
		for _, giftCard := range in.GiftCards {
			stats, _ := gcGet(in.GiftCardsBalanceMap, gcStr(giftCard, "code")).(*dal.Record)
			if stats == nil {
				return composer.StepResponse[any]{}, gcUndefinedProperty("balance")
			}
			if totals.Lte(gcGet(stats, "balance"), 0) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) has no balance", gcStr(giftCard, "code")))
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)
