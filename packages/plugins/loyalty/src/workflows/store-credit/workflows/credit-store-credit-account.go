package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/steps"
)

// StoreCreditAccountTransactionInput mirrors ModuleCreditStoreCreditAccount
// and ModuleDebitStoreCreditAccount.
type StoreCreditAccountTransactionInput struct {
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	Note        string  `json:"note,omitempty"`
	Reference   string  `json:"reference,omitempty"`
	ReferenceID string  `json:"reference_id,omitempty"`
}

// CreditStoreCreditAccountWorkflowInput mirrors
// CreditStoreCreditAccountWorkflowInput.
type CreditStoreCreditAccountWorkflowInput = StoreCreditAccountTransactionInput

// DebitStoreCreditAccountWorkflowInput mirrors
// DebitStoreCreditAccountWorkflowInput.
type DebitStoreCreditAccountWorkflowInput = StoreCreditAccountTransactionInput

func amountStep(name string) *composer.Step[StoreCreditAccountTransactionInput, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, input StoreCreditAccountTransactionInput) (composer.StepResponse[any], error) {
			if input.Amount <= 0 {
				return composer.StepResponse[any]{}, invalidData("Amount must be greater than 0")
			}
			return composer.NewStepResponse[any](nil), nil
		}, nil)
}

var (
	validateCreditStoreCreditAccountInputStep = amountStep("validate-store-credit-account-input")
	validateDebitStoreCreditAccountInputStep  = amountStep("validate-debit-store-credit-account-input")
)

// transactionData mirrors the transform building the credit or debit
// data: the account must exist (throwIfKeyNotFound).
func transactionData(e *composer.Execution, input StoreCreditAccountTransactionInput) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", input.AccountID)
	res, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: "store_credit_account", Fields: []string{"id", "code", "customer_id", "currency_code", "balance"}, Filters: f},
		Options:    joiner.Options{ThrowIfKeyNotFound: true},
	})
	if err != nil {
		return nil, err
	}
	account := first(res)
	if account == nil {
		return nil, undefinedProperty("id")
	}
	data := dal.NewRecord()
	data.Set("account_id", get(account, "id"))
	data.Set("amount", totals.Convert(input.Amount))
	data.Set("note", orEmpty(input.Note))
	data.Set("reference", orEmpty(input.Reference))
	data.Set("reference_id", orEmpty(input.ReferenceID))
	return data, nil
}

// CreditStoreCreditAccountWorkflow mirrors creditStoreCreditAccountWorkflow
// ("credit-store-credit-account"); it returns nothing.
var CreditStoreCreditAccountWorkflow = composer.CreateWorkflow("credit-store-credit-account",
	func(e *composer.Execution, input CreditStoreCreditAccountWorkflowInput) (any, error) {
		if _, err := validateCreditStoreCreditAccountInputStep.Run(e, input); err != nil {
			return nil, err
		}
		data, err := transactionData(e, input)
		if err != nil {
			return nil, err
		}
		_, err = steps.CreditAccountStep.Run(e, []*dal.Record{data})
		return nil, err
	})

// DebitStoreCreditAccountWorkflow mirrors debitStoreCreditAccountWorkflow
// ("debit-store-credit-account"); it returns nothing.
var DebitStoreCreditAccountWorkflow = composer.CreateWorkflow("debit-store-credit-account",
	func(e *composer.Execution, input DebitStoreCreditAccountWorkflowInput) (any, error) {
		if _, err := validateDebitStoreCreditAccountInputStep.Run(e, input); err != nil {
			return nil, err
		}
		data, err := transactionData(e, input)
		if err != nil {
			return nil, err
		}
		_, err = steps.DebitAccountStep.Run(e, []*dal.Record{data})
		return nil, err
	})
