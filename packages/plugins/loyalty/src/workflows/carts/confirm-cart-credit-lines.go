package carts

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/utils"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// ValidateStoreCreditAccountStepInput mirrors
// ValidateStoreCreditAccountStepInput.
type ValidateStoreCreditAccountStepInput struct {
	GiftCards []*dal.Record `json:"giftCards"`
	// GiftCardsBalanceMap maps gift card codes to their account stats.
	GiftCardsBalanceMap map[string]*dal.Record `json:"giftCardsBalanceMap"`
}

// ValidateStoreCreditAccountStep mirrors validateStoreCreditAccountStep
// ("validate-store-credit-account"): every gift card must have a balance.
var ValidateStoreCreditAccountStep = composer.CreateStep("validate-store-credit-account",
	func(ctx composer.StepExecutionContext, in ValidateStoreCreditAccountStepInput) (composer.StepResponse[any], error) {
		for _, gc := range in.GiftCards {
			code := fmt.Sprint(scGet(gc, "code"))
			stats := in.GiftCardsBalanceMap[code]
			if stats == nil {
				return composer.StepResponse[any]{}, scUndefined("balance")
			}
			if totals.Lte(scGet(stats, "balance"), 0) {
				return composer.StepResponse[any]{}, scInvalid(fmt.Sprintf("Gift card (%s) has no balance", code))
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateGiftCardsNotExpiredStepInput mirrors
// ValidateGiftCardsNotExpiredStepInput.
type ValidateGiftCardsNotExpiredStepInput struct {
	GiftCards []*dal.Record `json:"giftCards"`
}

// ValidateGiftCardsNotExpiredStep mirrors validateGiftCardsNotExpiredStep.
var ValidateGiftCardsNotExpiredStep = composer.CreateStep("validate-gift-cards-not-expired",
	func(ctx composer.StepExecutionContext, in ValidateGiftCardsNotExpiredStepInput) (composer.StepResponse[any], error) {
		for _, gc := range in.GiftCards {
			if utils.IsGiftCardExpired(scGet(gc, "expires_at")) {
				return composer.StepResponse[any]{}, scInvalid(fmt.Sprintf("Gift card (%v) has expired", scGet(gc, "code")))
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ConfirmCartCreditLinesWorkflowInput mirrors
// ConfirmCartCreditLinesWorkflowInput.
type ConfirmCartCreditLinesWorkflowInput struct {
	CartID string `json:"cart_id"`
}

// ConfirmCartCreditLinesWorkflow mirrors confirmCartCreditLinesWorkflow
// ("confirm-cart-credit-lines"): the store credit and gift card credit
// lines of the cart debit their accounts. It returns [].
var ConfirmCartCreditLinesWorkflow = composer.CreateWorkflow("confirm-cart-credit-lines",
	func(e *composer.Execution, input ConfirmCartCreditLinesWorkflowInput) ([]any, error) {
		cf := dal.NewRecord()
		cf.Set("id", input.CartID)
		cartQuery, err := commonsteps.UseQueryGraphStep.Config("get-existing-cart-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "cart", Fields: []string{
				"id", "customer_id", "currency_code",
				"credit_lines.id", "credit_lines.reference", "credit_lines.reference_id", "credit_lines.amount",
				"gift_cards.id", "gift_cards.code", "gift_cards.expires_at",
			}, Filters: cf},
			Options: joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		cart := scFirst(cartQuery)
		if cart == nil {
			return nil, scUndefined("gift_cards")
		}
		giftCards := scRecords(scGet(cart, "gift_cards"))
		if _, err := ValidateGiftCardsNotExpiredStep.Run(e, ValidateGiftCardsNotExpiredStepInput{GiftCards: giftCards}); err != nil {
			return nil, err
		}
		giftCardIDs := make([]any, 0, len(giftCards))
		for _, gc := range giftCards {
			giftCardIDs = append(giftCardIDs, scGet(gc, "id"))
		}

		lf := dal.NewRecord()
		lf.Set("gift_card_id", giftCardIDs)
		accountsQuery, err := commonsteps.UseQueryGraphStep.Config("get-store-credit-account-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "gift_card_store_credit_account", Fields: []string{"gift_card_id", "store_credit_account_id"}, Filters: lf},
		})
		if err != nil {
			return nil, err
		}
		accountsMap := map[string]any{}
		for _, row := range scRecords(accountsQuery.Data) {
			accountsMap[fmt.Sprint(scGet(row, "gift_card_id"))] = scGet(row, "store_credit_account_id")
		}

		debits := []*dal.Record{}
		for _, cl := range scRecords(scGet(cart, "credit_lines")) {
			ref := scGet(cl, "reference")
			if ref != "store-credit" && ref != "gift-card" {
				continue
			}
			accountID := accountsMap[fmt.Sprint(scGet(cl, "reference_id"))]
			if accountID == nil || accountID == "" {
				accountID = scGet(cl, "reference_id")
			}
			d := dal.NewRecord()
			d.Set("account_id", accountID)
			d.Set("amount", scGet(cl, "amount"))
			d.Set("reference", "cart")
			d.Set("reference_id", scGet(cart, "id"))
			if ref == "gift-card" {
				d.Set("note", "Gift card usage")
			} else {
				d.Set("note", "Store credit usage")
			}
			debits = append(debits, d)
		}
		if _, err := scworkflows.DebitAccountsWorkflow.RunAsStep(e, debits); err != nil {
			return nil, err
		}
		return []any{}, nil
	})
