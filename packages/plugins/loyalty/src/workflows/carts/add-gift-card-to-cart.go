package carts

import (
	"fmt"
	"strings"

	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/utils"
	gcsteps "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/steps"
)

// RetrieveGiftCardBalanceStepInput mirrors RetrieveGiftCardBalanceStepInput.
type RetrieveGiftCardBalanceStepInput struct {
	StoreCreditAccount *dal.Record `json:"storeCreditAccount"`
	GiftCard           *dal.Record `json:"giftCard"`
}

// RetrieveGiftCardBalanceStep mirrors retrieveGiftCardBalanceStep
// ("retrieve-gift-cards-balance"): the stats of the gift card's account,
// keyed by the gift card code.
var RetrieveGiftCardBalanceStep = composer.CreateStep("retrieve-gift-cards-balance",
	func(ctx composer.StepExecutionContext, in RetrieveGiftCardBalanceStepInput) (composer.StepResponse[*dal.Record], error) {
		if in.StoreCreditAccount == nil {
			return composer.StepResponse[*dal.Record]{}, gcUndefinedProperty("id")
		}
		stats, err := gcsteps.StoreCreditModule(ctx.Container).RetrieveAccountStats(ctx.Context, gcStr(in.StoreCreditAccount, "id"))
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		balances := dal.NewRecord()
		balances.Set(gcStr(in.GiftCard, "code"), stats)
		return composer.NewStepResponse(balances), nil
	}, nil)

// ValidateGiftCardExistsStepInput mirrors ValidateGiftCardExistsStepInput.
type ValidateGiftCardExistsStepInput struct {
	GiftCard *dal.Record `json:"giftCard,omitempty"`
	Input    gcCodeInput `json:"input"`
}

// gcCodeInput is the { code } part of the workflow input.
type gcCodeInput struct {
	Code   string `json:"code"`
	CartID string `json:"cart_id,omitempty"`
}

// gcValidateGiftCardExistsStep mirrors validateGiftCardStep of
// add-gift-card-to-cart.ts ("validate-gift-card"): the gift card exists.
var gcValidateGiftCardExistsStep = composer.CreateStep("validate-gift-card",
	func(ctx composer.StepExecutionContext, in ValidateGiftCardExistsStepInput) (composer.StepResponse[any], error) {
		if in.GiftCard == nil {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) not found", in.Input.Code))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateCartGiftCardStepInput mirrors ValidateCartGiftCardStepInput.
type ValidateCartGiftCardStepInput struct {
	Cart      *dal.Record   `json:"cart"`
	GiftCards []*dal.Record `json:"giftCards"`
}

// gcCartHasGiftCard mirrors cart.gift_cards.find((gc) =>
// gc.code.includes(code)).
func gcCartHasGiftCard(cart *dal.Record, code string) (bool, error) {
	if cart == nil {
		return false, gcUndefinedProperty("gift_cards")
	}
	list, ok := gcGet(cart, "gift_cards").([]any)
	if !ok {
		return false, fmt.Errorf("Cannot read properties of undefined (reading 'find')")
	}
	for _, gc := range gcRecords(list) {
		if strings.Contains(gcStr(gc, "code"), code) {
			return true, nil
		}
	}
	return false, nil
}

// ValidateCartGiftCardStep mirrors validateCartGiftCardStep
// ("validate-cart-gift-card"): a gift card is not applied twice, has not
// expired and is in the currency of the cart.
var ValidateCartGiftCardStep = composer.CreateStep("validate-cart-gift-card",
	func(ctx composer.StepExecutionContext, in ValidateCartGiftCardStepInput) (composer.StepResponse[any], error) {
		for _, giftCard := range in.GiftCards {
			code := gcStr(giftCard, "code")
			applied, err := gcCartHasGiftCard(in.Cart, code)
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			if applied {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) already applied to cart", code))
			}
			if utils.IsGiftCardExpired(gcGet(giftCard, "expires_at")) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) has expired", code))
			}
			if gcGet(giftCard, "currency_code") != gcGet(in.Cart, "currency_code") {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Gift card (%s) currency does not match cart currency", code))
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// AddGiftCardToCartWorkflowInput mirrors AddGiftCardToCartWorkflowInput.
type AddGiftCardToCartWorkflowInput struct {
	Code   string `json:"code"`
	CartID string `json:"cart_id"`
}

// gcCreditLinesForGiftCards mirrors the credit lines computed from the gift
// card balances (min(balance, remaining total) when positive); with
// decrement, the remaining total goes down with each credit line.
func gcCreditLinesForGiftCards(cart *dal.Record, giftCards []*dal.Record, balances *dal.Record, decrement bool) ([]any, error) {
	remaining := totals.Convert(gcGet(cart, "total"))
	lines := []any{}
	for _, giftCard := range giftCards {
		stats, _ := gcGet(balances, gcStr(giftCard, "code")).(*dal.Record)
		if stats == nil {
			return nil, gcUndefinedProperty("balance")
		}
		amount := totals.Min(gcGet(stats, "balance"), remaining)
		if !totals.Gt(amount, 0) {
			continue
		}
		lines = append(lines, gcRecord(
			"cart_id", gcGet(cart, "id"),
			"amount", gcToNumber(amount),
			"reference", "gift-card",
			"reference_id", gcGet(giftCard, "id"),
			"metadata", dal.NewRecord(),
		))
		if decrement {
			remaining = totals.Sub(remaining, amount)
		}
	}
	return lines, nil
}

// AddGiftCardToCartWorkflow mirrors addGiftCardToCartWorkflow
// ("add-gift-card-to-cart"): a credit line for the gift card balance (up to
// the cart total), the gift card linked to the cart, the cart refreshed. Its
// "validate" hook runs after the built-in validations.
var AddGiftCardToCartWorkflow = composer.CreateWorkflow("add-gift-card-to-cart",
	func(e *composer.Execution, input AddGiftCardToCartWorkflowInput) ([]any, error) {
		cartRows, err := gcQueryGraph(e, "get-cart-query", "cart", []string{"id", "currency_code", "total", "gift_cards.code"}, gcRecord("id", input.CartID), false)
		if err != nil {
			return nil, err
		}
		cart := gcFirst(cartRows)
		giftCardRows, err := gcQueryGraph(e, "get-gift-card-query", "gift_card", []string{"id", "code", "status", "currency_code", "expires_at"}, gcRecord("code", input.Code), false)
		if err != nil {
			return nil, err
		}
		giftCard := gcFirst(giftCardRows)
		codeInput := gcCodeInput{Code: input.Code, CartID: input.CartID}
		if _, err := gcValidateGiftCardExistsStep.Run(e, ValidateGiftCardExistsStepInput{GiftCard: giftCard, Input: codeInput}); err != nil {
			return nil, err
		}
		giftCards := gcRecords(giftCardRows)
		if _, err := ValidateCartGiftCardStep.Run(e, ValidateCartGiftCardStepInput{Cart: cart, GiftCards: giftCards}); err != nil {
			return nil, err
		}
		hookInput := gcRecord("input", gcRecord("code", input.Code, "cart_id", input.CartID), "cart", cart, "giftCards", giftCardRows)
		if err := composer.CreateHook(e, "validate", hookInput); err != nil {
			return nil, err
		}
		linkRows, err := gcQueryGraph(e, "get-gift-card-store-credit-account-query", "gift_card_store_credit_account",
			[]string{"id", "store_credit_account_id"}, gcRecord("gift_card_id", gcGet(giftCard, "id")), false)
		if err != nil {
			return nil, err
		}
		link := gcFirst(linkRows)
		if link == nil {
			return nil, gcUndefinedProperty("store_credit_account_id")
		}
		accountRows, err := gcQueryGraph(e, "get-store-credit-account-query", "store_credit_account", []string{"id", "balance"},
			gcRecord("id", gcGet(link, "store_credit_account_id"), "currency_code", gcGet(cart, "currency_code")), false)
		if err != nil {
			return nil, err
		}
		balances, err := RetrieveGiftCardBalanceStep.Run(e, RetrieveGiftCardBalanceStepInput{StoreCreditAccount: gcFirst(accountRows), GiftCard: giftCard})
		if err != nil {
			return nil, err
		}
		if _, err := ValidateGiftCardBalancesStep.Run(e, ValidateGiftCardBalancesStepInput{GiftCards: giftCards, GiftCardsBalanceMap: balances}); err != nil {
			return nil, err
		}
		toCreate, err := gcCreditLinesForGiftCards(cart, giftCards, balances, false)
		if err != nil {
			return nil, err
		}
		creditLines, err := cartworkflows.CreateCartCreditLinesWorkflow.RunAsStep(e, toCreate)
		if err != nil {
			return nil, err
		}
		links := []*dal.Record{}
		for _, creditLine := range gcRecords(creditLines) {
			if gcStr(creditLine, "reference") != "gift-card" {
				continue
			}
			links = append(links, gcRecord(
				utilsmodulessdk.CART, gcRecord("cart_id", gcGet(cart, "id")),
				loyalty.ServiceName, gcRecord("gift_card_id", gcGet(creditLine, "reference_id")),
			))
		}
		if _, err := commonworkflows.CreateLinksWorkflow.RunAsStep(e, links); err != nil {
			return nil, err
		}
		if _, err := cartworkflows.RefreshCartItemsWorkflow.RunAsStep(e, cartworkflows.RefreshCartItemsInput{CartID: input.CartID}); err != nil {
			return nil, err
		}
		return creditLines, nil
	})
