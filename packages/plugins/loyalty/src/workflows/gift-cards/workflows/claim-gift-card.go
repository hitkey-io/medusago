package workflows

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

// ClaimGiftCardWorkflowInput mirrors ClaimGiftCardWorkflowInput.
type ClaimGiftCardWorkflowInput struct {
	Code       string `json:"code"`
	CustomerID string `json:"customer_id"`
}

// ValidateClaimGiftCardInputStepInput is the input of
// validateClaimGiftCardInputStep.
type ValidateClaimGiftCardInputStepInput struct {
	GiftCard *dal.Record `json:"giftCard"`
	Customer *dal.Record `json:"customer"`
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

// ValidateClaimGiftCardInputStep mirrors validateClaimGiftCardInputStep
// ("validate-claim-gift-card-input").
var ValidateClaimGiftCardInputStep = composer.CreateStep("validate-claim-gift-card-input",
	func(ctx composer.StepExecutionContext, in ValidateClaimGiftCardInputStepInput) (composer.StepResponse[any], error) {
		if in.GiftCard == nil {
			return composer.StepResponse[any]{}, fmt.Errorf("Cannot read properties of undefined (reading 'store_credit_account')")
		}
		account, _ := get(in.GiftCard, "store_credit_account").(*dal.Record)
		if !truthy(get(account, "id")) {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Gift card does not have a store credit account")
		}
		if !truthy(get(account, "code")) {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Gift card does not have a store credit account code")
		}
		if in.Customer == nil {
			return composer.StepResponse[any]{}, fmt.Errorf("Cannot read properties of undefined (reading 'has_account')")
		}
		if !truthy(get(in.Customer, "has_account")) {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Only customers with an account can claim a gift card")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ClaimGiftCardWorkflow mirrors claimGiftCardWorkflow ("claim-gift-card"):
// the balance of the gift card's store credit account goes to the
// customer's account.
var ClaimGiftCardWorkflow = composer.CreateWorkflow("claim-gift-card",
	func(e *composer.Execution, input ClaimGiftCardWorkflowInput) (any, error) {
		giftCards, err := queryGraph(e, "gift-card-data", "gift_card",
			[]string{"id", "code", "customer_id", "currency_code", "status", "store_credit_account.id", "store_credit_account.code"},
			filter("code", input.Code), false)
		if err != nil {
			return nil, err
		}
		giftCard := first(giftCards)
		customers, err := queryGraph(e, "customer-data", "customer", []string{"id", "email", "has_account"}, filter("id", input.CustomerID), false)
		if err != nil {
			return nil, err
		}
		customer := first(customers)
		if _, err := ValidateClaimGiftCardInputStep.Run(e, ValidateClaimGiftCardInputStepInput{GiftCard: giftCard, Customer: customer}); err != nil {
			return nil, err
		}
		account, _ := get(giftCard, "store_credit_account").(*dal.Record)
		if _, err := scworkflows.ClaimStoreCreditAccountWorkflow.RunAsStep(e, scworkflows.ClaimStoreCreditAccountWorkflowInput{
			Code:       str(account, "code"),
			CustomerID: str(customer, "id"),
		}); err != nil {
			return nil, err
		}
		return nil, nil
	})
