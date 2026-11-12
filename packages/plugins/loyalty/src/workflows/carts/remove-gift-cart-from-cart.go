package carts

import (
	"fmt"

	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
)

// ValidateGiftCardInCartStepInput mirrors ValidateGiftCardInCartStepInput.
type ValidateGiftCardInCartStepInput struct {
	Cart     *dal.Record `json:"cart"`
	GiftCard *dal.Record `json:"giftCard"`
}

// ValidateGiftCardInCartStep mirrors validateGiftCardInCartStep
// ("validate-gift-card-in-cart"): the gift card is applied to the cart.
var ValidateGiftCardInCartStep = composer.CreateStep("validate-gift-card-in-cart",
	func(ctx composer.StepExecutionContext, in ValidateGiftCardInCartStepInput) (composer.StepResponse[any], error) {
		code := gcStr(in.GiftCard, "code")
		applied, err := gcCartHasGiftCard(in.Cart, code)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if !applied {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) not found in cart", code))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateGiftCardStepInput mirrors ValidateGiftCardStepInput.
type ValidateGiftCardStepInput struct {
	Cart     *dal.Record `json:"cart"`
	GiftCard *dal.Record `json:"giftCard,omitempty"`
	Input    gcCodeInput `json:"input"`
}

// ValidateGiftCardStep mirrors validateGiftCardStep of
// remove-gift-cart-from-cart.ts ("validate-gift-card"): the code is applied
// to the cart and the gift card exists.
var ValidateGiftCardStep = composer.CreateStep("validate-gift-card",
	func(ctx composer.StepExecutionContext, in ValidateGiftCardStepInput) (composer.StepResponse[any], error) {
		found := false
		for _, gc := range gcRecords(gcGet(in.Cart, "gift_cards")) {
			if gcStr(gc, "code") == in.Input.Code {
				found = true
				break
			}
		}
		if !found {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) not found in cart", in.Input.Code))
		}
		if in.GiftCard == nil {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) not found", in.Input.Code))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RemoveGiftCardFromCartWorkflowInput mirrors RemoveGiftCardFromCartWorkflowInput.
type RemoveGiftCardFromCartWorkflowInput struct {
	Code   string `json:"code"`
	CartID string `json:"cart_id"`
}

// RemoveGiftCardFromCartWorkflow mirrors removeGiftCardFromCartWorkflow
// ("remove-gift-card-from-cart"): the gift card credit lines deleted, the
// link to the cart dismissed, the cart refreshed.
var RemoveGiftCardFromCartWorkflow = composer.CreateWorkflow("remove-gift-card-from-cart",
	func(e *composer.Execution, input RemoveGiftCardFromCartWorkflowInput) (any, error) {
		cartRows, err := gcQueryGraph(e, "get-cart-query", "cart", []string{
			"id", "credit_lines.id", "credit_lines.reference", "credit_lines.reference_id", "gift_cards.id", "gift_cards.code",
		}, gcRecord("id", input.CartID), true)
		if err != nil {
			return nil, err
		}
		cart := gcFirst(cartRows)
		giftCardRows, err := gcQueryGraph(e, "get-gift-card-query", "gift_card", []string{"id", "code"}, gcRecord("code", input.Code), false)
		if err != nil {
			return nil, err
		}
		giftCard := gcFirst(giftCardRows)
		if _, err := ValidateGiftCardStep.Run(e, ValidateGiftCardStepInput{Cart: cart, GiftCard: giftCard, Input: gcCodeInput{Code: input.Code, CartID: input.CartID}}); err != nil {
			return nil, err
		}
		if _, err := ValidateGiftCardInCartStep.Run(e, ValidateGiftCardInCartStepInput{Cart: cart, GiftCard: giftCard}); err != nil {
			return nil, err
		}
		creditLineIDs := []any{}
		for _, creditLine := range gcRecords(gcGet(cart, "credit_lines")) {
			if gcStr(creditLine, "reference") == "gift-card" && gcGet(creditLine, "reference_id") == gcGet(giftCard, "id") {
				creditLineIDs = append(creditLineIDs, gcGet(creditLine, "id"))
			}
		}
		if _, err := cartworkflows.DeleteCartCreditLinesWorkflow.RunAsStep(e, cartworkflows.DeleteCartCreditLinesInput{ID: creditLineIDs}); err != nil {
			return nil, err
		}
		if _, err := commonworkflows.DismissLinksWorkflow.RunAsStep(e, []*dal.Record{gcRecord(
			utilsmodulessdk.CART, gcRecord("cart_id", gcGet(cart, "id")),
			loyalty.ServiceName, gcRecord("gift_card_id", gcGet(giftCard, "id")),
		)}); err != nil {
			return nil, err
		}
		if _, err := cartworkflows.RefreshCartItemsWorkflow.RunAsStep(e, cartworkflows.RefreshCartItemsInput{CartID: input.CartID}); err != nil {
			return nil, err
		}
		return nil, nil
	})
