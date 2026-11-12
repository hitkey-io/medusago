package workflows

import (
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/steps"
)

// ClaimStoreCreditAccountWorkflowInput mirrors
// ClaimStoreCreditAccountWorkflowInput.
type ClaimStoreCreditAccountWorkflowInput struct {
	Code       string `json:"code"`
	CustomerID string `json:"customer_id"`
}

// ValidateClaimStoreCreditAccountInputStepInput mirrors
// ValidateClaimStoreCreditAccountInputStepInput.
type ValidateClaimStoreCreditAccountInputStepInput struct {
	Input    ClaimStoreCreditAccountWorkflowInput `json:"input"`
	Customer *dal.Record                          `json:"customer"`
}

// ValidateClaimStoreCreditAccountInputStep mirrors
// validateClaimStoreCreditAccountInputStep.
var ValidateClaimStoreCreditAccountInputStep = composer.CreateStep("validate-claim-store-credit-account-input",
	func(ctx composer.StepExecutionContext, in ValidateClaimStoreCreditAccountInputStepInput) (composer.StepResponse[any], error) {
		if !cartutils.IsPresent(in.Input.Code) {
			return composer.StepResponse[any]{}, invalidData("Code is required to claim a store credit account")
		}
		if in.Input.CustomerID == "" {
			return composer.StepResponse[any]{}, invalidData("Customer Id is required to claim a store credit account")
		}
		if in.Customer == nil {
			return composer.StepResponse[any]{}, undefinedProperty("has_account")
		}
		if b, _ := get(in.Customer, "has_account").(bool); !b {
			return composer.StepResponse[any]{}, invalidData("Only customers with an account can claim a store credit account")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ValidateSourceStoreCreditAccountsStepInput is the input of
// validateSourceStoreCreditAccountsStep.
type ValidateSourceStoreCreditAccountsStepInput struct {
	SourceStoreCreditAccount *dal.Record `json:"sourceStoreCreditAccount"`
	TargetStoreCreditAccount *dal.Record `json:"targetStoreCreditAccount"`
}

// ValidateSourceStoreCreditAccountsStep mirrors
// validateSourceStoreCreditAccountsStep ("validate-source-store-credit-account").
var ValidateSourceStoreCreditAccountsStep = composer.CreateStep("validate-source-store-credit-account",
	func(ctx composer.StepExecutionContext, in ValidateSourceStoreCreditAccountsStepInput) (composer.StepResponse[any], error) {
		source, target := in.SourceStoreCreditAccount, in.TargetStoreCreditAccount
		if source == nil {
			return composer.StepResponse[any]{}, undefinedProperty("id")
		}
		if target == nil {
			return composer.StepResponse[any]{}, undefinedProperty("id")
		}
		if get(source, "id") == get(target, "id") {
			return composer.StepResponse[any]{}, invalidData("Customer already owns the store credit account")
		}
		if cid := get(source, "customer_id"); cid != nil && cid != "" {
			return composer.StepResponse[any]{}, invalidData("Cannot claim a store credit account that belongs to a customer")
		}
		if totals.Lte(get(source, "balance"), 0) {
			return composer.StepResponse[any]{}, invalidData("Cannot claim a store credit account with no balance")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

var accountClaimFields = []string{"id", "code", "customer_id", "currency_code", "balance"}

// ClaimStoreCreditAccountWorkflow mirrors claimStoreCreditAccountWorkflow
// ("claim-store-credit-account"): the balance of an anonymous account moves
// to the customer's account in its currency (created when missing).
var ClaimStoreCreditAccountWorkflow = composer.CreateWorkflow("claim-store-credit-account",
	func(e *composer.Execution, input ClaimStoreCreditAccountWorkflowInput) (any, error) {
		sf := dal.NewRecord()
		sf.Set("code", input.Code)
		sourceData, err := commonsteps.UseQueryGraphStep.Config("source-store-credit-account-data", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "store_credit_account", Fields: accountClaimFields, Filters: sf},
		})
		if err != nil {
			return nil, err
		}
		source := first(sourceData)

		cf := dal.NewRecord()
		cf.Set("id", input.CustomerID)
		customerData, err := commonsteps.UseQueryGraphStep.Config("customer-data", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "customer", Fields: []string{"id", "email", "has_account"}, Filters: cf},
		})
		if err != nil {
			return nil, err
		}
		customer := first(customerData)

		if _, err := ValidateClaimStoreCreditAccountInputStep.Run(e, ValidateClaimStoreCreditAccountInputStepInput{Input: input, Customer: customer}); err != nil {
			return nil, err
		}

		// accountCurrencyCode = sourceStoreCreditAccount.currency_code
		if source == nil {
			return nil, undefinedProperty("currency_code")
		}
		ef := dal.NewRecord()
		ef.Set("customer_id", input.CustomerID)
		ef.Set("currency_code", get(source, "currency_code"))
		existingData, err := commonsteps.UseQueryGraphStep.Config("existing-customer-store-credit-account-data", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "store_credit_account", Fields: accountClaimFields, Filters: ef},
		})
		if err != nil {
			return nil, err
		}
		target := first(existingData)

		// when("store-account-does-not-exist")
		if target == nil {
			account := dal.NewRecord()
			account.Set("customer_id", input.CustomerID)
			account.Set("currency_code", get(source, "currency_code"))
			created, err := steps.CreateStoreCreditAccountsStep.Run(e, []*dal.Record{account})
			if err != nil {
				return nil, err
			}
			if len(created) > 0 {
				target = created[0]
			}
		}

		balanceToTransfer := totals.Convert(get(source, "balance"))

		if _, err := ValidateSourceStoreCreditAccountsStep.Run(e, ValidateSourceStoreCreditAccountsStepInput{SourceStoreCreditAccount: source, TargetStoreCreditAccount: target}); err != nil {
			return nil, err
		}

		debit := dal.NewRecord()
		debit.Set("account_id", get(source, "id"))
		debit.Set("amount", balanceToTransfer)
		debit.Set("reference", "store-credit")
		debit.Set("reference_id", get(target, "id"))
		if _, err := steps.DebitAccountStep.Run(e, []*dal.Record{debit}); err != nil {
			return nil, err
		}

		credit := dal.NewRecord()
		credit.Set("account_id", get(target, "id"))
		credit.Set("amount", balanceToTransfer)
		credit.Set("reference", "store-credit")
		credit.Set("reference_id", get(source, "id"))
		if _, err := steps.CreditAccountStep.Run(e, []*dal.Record{credit}); err != nil {
			return nil, err
		}
		return nil, nil
	})
