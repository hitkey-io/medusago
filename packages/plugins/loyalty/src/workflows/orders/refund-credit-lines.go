package orders

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	scsteps "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/steps"
	scworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/workflows"
)

func scGet(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

func scFirst(res commonsteps.QueryGraphResult) *dal.Record {
	switch t := res.Data.(type) {
	case []any:
		if len(t) > 0 {
			r, _ := t[0].(*dal.Record)
			return r
		}
	case []*dal.Record:
		if len(t) > 0 {
			return t[0]
		}
	}
	return nil
}

// scNegativeCreditLines mirrors the credit lines with a negative amount.
func scNegativeCreditLines(lines []*dal.Record) []*dal.Record {
	out := []*dal.Record{}
	for _, cl := range lines {
		if totals.Lt(scGet(cl, "amount"), 0) {
			out = append(out, cl)
		}
	}
	return out
}

type scValidateCustomerStepInput struct {
	Customer            *dal.Record   `json:"customer"`
	NegativeCreditLines []*dal.Record `json:"negativeCreditLines"`
}

// scValidateCustomerStep mirrors validateCustomerStep ("validate-customer"):
// only registered customers get store credit refunds.
var scValidateCustomerStep = composer.CreateStep("validate-customer",
	func(ctx composer.StepExecutionContext, in scValidateCustomerStepInput) (composer.StepResponse[any], error) {
		hasAccount, _ := scGet(in.Customer, "has_account").(bool)
		if !hasAccount && len(in.NegativeCreditLines) > 0 {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Store credit refunds can only be issued to registered customers")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreditAccountTransactionsStepInput mirrors CreditAccountTransactionsStepInput.
type CreditAccountTransactionsStepInput struct {
	Order              *dal.Record   `json:"order"`
	StoreCreditAccount *dal.Record   `json:"storeCreditAccount"`
	CreditLines        []*dal.Record `json:"creditLines"`
}

// CreditAccountTransactionsStep mirrors creditAccountTransactionsStep
// ("credit-account-transactions"): the negative credit lines become one
// credit of the customer's account in the order currency (created, without
// a code, when missing).
var CreditAccountTransactionsStep = composer.CreateStep("credit-account-transactions",
	func(ctx composer.StepExecutionContext, in CreditAccountTransactionsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		total := totals.Convert(0)
		for _, cl := range scNegativeCreditLines(in.CreditLines) {
			total = totals.Add(total, totals.Mult(scGet(cl, "amount"), -1))
		}
		account := in.StoreCreditAccount
		if account == nil {
			data := dal.NewRecord()
			data.Set("customer_id", scGet(in.Order, "customer_id"))
			data.Set("currency_code", scGet(in.Order, "currency_code"))
			created, err := scsteps.Service(ctx.Container).CreateStoreCreditAccounts(ctx.Context, []*dal.Record{data})
			if err != nil {
				return composer.StepResponse[[]*dal.Record]{}, err
			}
			if len(created) > 0 {
				account = created[0]
			}
		}
		t := dal.NewRecord()
		t.Set("account_id", scGet(account, "id"))
		t.Set("amount", total)
		t.Set("reference", "order")
		t.Set("reference_id", scGet(in.Order, "id"))
		t.Set("note", "Store credit refund")
		return composer.NewStepResponse([]*dal.Record{t}), nil
	}, nil)

// RefundCreditLinesWorkflowInput mirrors RefundCreditLinesWorkflowInput.
type RefundCreditLinesWorkflowInput struct {
	OrderID     string        `json:"order_id"`
	CreditLines []*dal.Record `json:"credit_lines"`
}

// RefundCreditLinesWorkflow mirrors refundCreditLinesWorkflow
// ("refund-credit-lines"): negative order credit lines are refunded to the
// customer's store credit account.
var RefundCreditLinesWorkflow = composer.CreateWorkflow("refund-credit-lines",
	func(e *composer.Execution, input RefundCreditLinesWorkflowInput) (any, error) {
		of := dal.NewRecord()
		of.Set("id", input.OrderID)
		orderQuery, err := commonsteps.UseQueryGraphStep.Config("get-cart-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: []string{"id", "customer.*", "customer_id", "currency_code"}, Filters: of},
		})
		if err != nil {
			return nil, err
		}
		order := scFirst(orderQuery)

		cf := dal.NewRecord()
		cf.Set("id", scGet(order, "customer_id"))
		customerQuery, err := commonsteps.UseQueryGraphStep.Config("get-customer-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "customer", Fields: []string{"id", "email", "has_account"}, Filters: cf},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		customer := scFirst(customerQuery)

		negative := scNegativeCreditLines(input.CreditLines)
		if _, err := scValidateCustomerStep.Run(e, scValidateCustomerStepInput{Customer: customer, NegativeCreditLines: negative}); err != nil {
			return nil, err
		}

		af := dal.NewRecord()
		af.Set("customer_id", scGet(customer, "id"))
		af.Set("currency_code", scGet(order, "currency_code"))
		accountsQuery, err := commonsteps.UseQueryGraphStep.Config("get-store-credit-accounts-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "store_credit_account", Fields: []string{"id", "customer_id", "balance", "credits", "debits"}, Filters: af},
		})
		if err != nil {
			return nil, err
		}
		account := scFirst(accountsQuery)

		if len(negative) > 0 {
			transactions, err := CreditAccountTransactionsStep.Run(e, CreditAccountTransactionsStepInput{Order: order, StoreCreditAccount: account, CreditLines: negative})
			if err != nil {
				return nil, err
			}
			if _, err := scworkflows.CreditAccountsWorkflow.RunAsStep(e, transactions); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})
