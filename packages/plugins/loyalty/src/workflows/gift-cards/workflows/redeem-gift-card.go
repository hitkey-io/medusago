package workflows

import (
	"fmt"

	commonworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
	storecredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit"
	scsteps "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/steps"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// RedeemGiftCardWorkflowInput mirrors RedeemGiftCardWorkflowInput.
type RedeemGiftCardWorkflowInput struct {
	GiftCardID string `json:"gift_card_id"`
}

// ValidateGiftCardRedeemStepInput mirrors ValidateGiftCardRedeemStepInput.
type ValidateGiftCardRedeemStepInput struct {
	GiftCard                   *dal.Record                 `json:"giftCard"`
	GiftCardStoreCreditAccount *dal.Record                 `json:"giftCardStoreCreditAccount,omitempty"`
	Input                      RedeemGiftCardWorkflowInput `json:"input"`
}

// ValidateGiftCardRedeemStep mirrors validateGiftCardRedeemStep
// ("validate-gift-card-redeem").
var ValidateGiftCardRedeemStep = composer.CreateStep("validate-gift-card-redeem",
	func(ctx composer.StepExecutionContext, in ValidateGiftCardRedeemStepInput) (composer.StepResponse[any], error) {
		if in.GiftCard == nil {
			return composer.StepResponse[any]{}, fmt.Errorf("Cannot read properties of undefined (reading 'status')")
		}
		if str(in.GiftCard, "status") == "redeemed" {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Gift card is already redeemed")
		}
		if in.GiftCardStoreCreditAccount != nil {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Gift card already has a store credit account")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RedeemGiftCardWorkflow mirrors redeemGiftCardWorkflow ("redeem-gift-card"):
// an anonymous store credit account linked to the gift card and credited
// with its value, the gift card marked as redeemed; the result is the
// account.
var RedeemGiftCardWorkflow = composer.CreateWorkflow("redeem-gift-card",
	func(e *composer.Execution, input RedeemGiftCardWorkflowInput) (*dal.Record, error) {
		rows, err := queryGraph(e, "get-gift-card-query", "gift_card", []string{"id", "code", "status", "value", "currency_code"}, filter("id", input.GiftCardID), true)
		if err != nil {
			return nil, err
		}
		giftCard := first(rows)
		linkRows, err := queryGraph(e, "get-gift-card-store-credit-account-query", "gift_card_store_credit_account",
			[]string{"id", "store_credit_account_id"}, filter("gift_card_id", get(giftCard, "id")), false)
		if err != nil {
			return nil, err
		}
		if _, err := ValidateGiftCardRedeemStep.Run(e, ValidateGiftCardRedeemStepInput{GiftCard: giftCard, GiftCardStoreCreditAccount: first(linkRows), Input: input}); err != nil {
			return nil, err
		}
		accounts, err := scsteps.CreateStoreCreditAccountsStep.Run(e, []*dal.Record{record("currency_code", get(giftCard, "currency_code"))})
		if err != nil {
			return nil, err
		}
		var account *dal.Record
		if len(accounts) > 0 {
			account = accounts[0]
		}
		links := []*dal.Record{record(
			loyalty.ServiceName, record("gift_card_id", get(giftCard, "id")),
			storecredit.ServiceName, record("store_credit_account_id", get(account, "id")),
		)}
		if _, err := commonworkflows.CreateLinksWorkflow.RunAsStep(e, links); err != nil {
			return nil, err
		}
		if _, err := scworkflows.CreditAccountsWorkflow.RunAsStep(e, []*dal.Record{record(
			"account_id", get(account, "id"),
			"amount", get(giftCard, "value"),
			"note", "Gift card redemption",
			"reference", "gift_card",
			"reference_id", get(giftCard, "code"),
		)}); err != nil {
			return nil, err
		}
		if _, err := UpdateGiftCardsWorkflow.RunAsStep(e, []*dal.Record{record("id", get(giftCard, "id"), "status", "redeemed")}); err != nil {
			return nil, err
		}
		accountRows, err := queryGraph(e, "get-account-query", "store_credit_account", []string{"id", "debits", "credits", "code", "transactions.*"}, filter("id", get(account, "id")), true)
		if err != nil {
			return nil, err
		}
		return first(accountRows), nil
	})

func init() {
	// Run through the workflow engine (throwOnError false), a failed run
	// still resolves the return transform: accountQuery.data[0] of a query
	// step that never ran throws, and that TypeError replaces the step
	// error.
	RedeemGiftCardWorkflow.SetEngineFailure(func(error) error {
		return fmt.Errorf("Cannot read properties of undefined (reading 'data')")
	})
}
