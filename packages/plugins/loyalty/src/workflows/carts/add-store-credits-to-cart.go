package carts

import (
	"fmt"

	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func scInvalid(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func scGet(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

// scFirst mirrors <query>.data[0] (nil when empty).
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

func scRecords(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func scNumber(bn *totals.BN) float64 {
	return totals.JSONNumber(bn).(float64)
}

func scUndefined(key string) error {
	return fmt.Errorf("Cannot read properties of undefined (reading '%s')", key)
}

// ValidateCustomerStoreCreditAccountStepInput mirrors
// ValidateCustomerStoreCreditAccountStepInput.
type ValidateCustomerStoreCreditAccountStepInput struct {
	StoreCreditAccount *dal.Record `json:"storeCreditAccount"`
}

// ValidateCustomerStoreCreditAccountStep mirrors
// validateCustomerStoreCreditAccountStep.
var ValidateCustomerStoreCreditAccountStep = composer.CreateStep("validate-customer-store-credit-account",
	func(ctx composer.StepExecutionContext, in ValidateCustomerStoreCreditAccountStepInput) (composer.StepResponse[any], error) {
		if in.StoreCreditAccount == nil {
			return composer.StepResponse[any]{}, scInvalid("Store credit account not found for the customer on the cart in that currency.")
		}
		if totals.Lte(scGet(in.StoreCreditAccount, "balance"), 0) {
			return composer.StepResponse[any]{}, scInvalid("Store credit account has no balance")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// AddStoreCreditsToCartWorkflowInput mirrors
// AddStoreCreditsToCartWorkflowInput.
type AddStoreCreditsToCartWorkflowInput struct {
	CartID string `json:"cart_id"`
	// Amount is the amount to use (undefined: the whole balance).
	Amount     *float64 `json:"amount,omitempty"`
	CustomerID *string  `json:"customer_id,omitempty"`
}

// ValidateCartStepInput mirrors ValidateCartStepInput.
type ValidateCartStepInput struct {
	Cart  *dal.Record                        `json:"cart"`
	Input AddStoreCreditsToCartWorkflowInput `json:"input"`
}

// ValidateCartStep mirrors validateCartStep ("validate-cart").
var ValidateCartStep = composer.CreateStep("validate-cart",
	func(ctx composer.StepExecutionContext, in ValidateCartStepInput) (composer.StepResponse[any], error) {
		cart := in.Cart
		if cart == nil {
			return composer.StepResponse[any]{}, scInvalid(fmt.Sprintf("Cart with id %s not found", in.Input.CartID))
		}
		customerID := scGet(cart, "customer_id")
		if customerID == nil || customerID == "" {
			return composer.StepResponse[any]{}, scInvalid("Cart's customer must be set to add store credits")
		}
		if in.Input.CustomerID != nil && customerID != *in.Input.CustomerID {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Store credits can only be added to a cart that belongs to the authenticated customer")
		}
		if cc := scGet(cart, "currency_code"); cc == nil || cc == "" {
			return composer.StepResponse[any]{}, scInvalid("Cart's currency must be set to add store credits")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ComputeCreditLineActionsStepInput mirrors ComputeCreditLineActionsStepInput.
type ComputeCreditLineActionsStepInput struct {
	StoreCreditAccount *dal.Record                        `json:"storeCreditAccount"`
	Cart               *dal.Record                        `json:"cart"`
	Input              AddStoreCreditsToCartWorkflowInput `json:"input"`
}

// CreditLineActions is the output of computeCreditLineActionsStep.
type CreditLineActions struct {
	CreditLinesToCreate []any `json:"creditLinesToCreate"`
	CreditLinesToDelete []any `json:"creditLinesToDelete"`
}

// ComputeCreditLineActionsStep mirrors computeCreditLineActionsStep: the
// store credit lines of the cart are replaced by one for the amount (the
// whole balance by default), capped at the cart total.
var ComputeCreditLineActionsStep = composer.CreateStep("compute-credit-line-actions",
	func(ctx composer.StepExecutionContext, in ComputeCreditLineActionsStepInput) (composer.StepResponse[CreditLineActions], error) {
		out := CreditLineActions{CreditLinesToCreate: []any{}, CreditLinesToDelete: []any{}}
		for _, cl := range scRecords(scGet(in.Cart, "credit_lines")) {
			if scGet(cl, "reference") == "store-credit" {
				out.CreditLinesToDelete = append(out.CreditLinesToDelete, scGet(cl, "id"))
			}
		}
		balance := scGet(in.StoreCreditAccount, "balance")
		var amount *totals.BN
		if in.Input.Amount != nil {
			amount = totals.Convert(*in.Input.Amount)
			if totals.Gt(amount, balance) {
				return composer.StepResponse[CreditLineActions]{}, scInvalid("Amount is greater than the store credit account balance")
			}
		} else {
			amount = totals.Convert(balance)
		}
		if totals.Gt(amount, 0) {
			line := dal.NewRecord()
			line.Set("cart_id", scGet(in.Cart, "id"))
			line.Set("amount", scNumber(totals.Min(amount, scGet(in.Cart, "total"))))
			line.Set("reference", "store-credit")
			line.Set("reference_id", scGet(in.StoreCreditAccount, "id"))
			line.Set("metadata", dal.NewRecord())
			out.CreditLinesToCreate = append(out.CreditLinesToCreate, line)
		}
		return composer.NewStepResponse(out), nil
	}, nil)

// AddStoreCreditsToCartWorkflow mirrors addStoreCreditsToCartWorkflow
// ("add-store-credits-to-cart"): it returns the created credit lines.
var AddStoreCreditsToCartWorkflow = composer.CreateWorkflow("add-store-credits-to-cart",
	func(e *composer.Execution, input AddStoreCreditsToCartWorkflowInput) ([]any, error) {
		cf := dal.NewRecord()
		cf.Set("id", input.CartID)
		cartQuery, err := commonsteps.UseQueryGraphStep.Config("get-cart-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "cart", Fields: []string{"id", "currency_code", "total", "customer_id", "credit_lines.*"}, Filters: cf},
		})
		if err != nil {
			return nil, err
		}
		cart := scFirst(cartQuery)
		if _, err := ValidateCartStep.Run(e, ValidateCartStepInput{Cart: cart, Input: input}); err != nil {
			return nil, err
		}

		af := dal.NewRecord()
		af.Set("customer_id", scGet(cart, "customer_id"))
		af.Set("currency_code", scGet(cart, "currency_code"))
		accountQuery, err := commonsteps.UseQueryGraphStep.Config("get-store-credit-account-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "store_credit_account", Fields: []string{"id", "balance"}, Filters: af},
		})
		if err != nil {
			return nil, err
		}
		account := scFirst(accountQuery)
		if _, err := ValidateCustomerStoreCreditAccountStep.Run(e, ValidateCustomerStoreCreditAccountStepInput{StoreCreditAccount: account}); err != nil {
			return nil, err
		}

		actions, err := ComputeCreditLineActionsStep.Run(e, ComputeCreditLineActionsStepInput{StoreCreditAccount: account, Cart: cart, Input: input})
		if err != nil {
			return nil, err
		}

		var created []any
		err = composer.Parallelize(
			func() error {
				_, err := cartworkflows.DeleteCartCreditLinesWorkflow.RunAsStep(e, cartworkflows.DeleteCartCreditLinesInput{ID: actions.CreditLinesToDelete})
				return err
			},
			func() error {
				var err error
				created, err = cartworkflows.CreateCartCreditLinesWorkflow.RunAsStep(e, actions.CreditLinesToCreate)
				return err
			},
		)
		if err != nil {
			return nil, err
		}

		if _, err := cartworkflows.RefreshCartItemsWorkflow.RunAsStep(e, cartworkflows.RefreshCartItemsInput{CartID: input.CartID}); err != nil {
			return nil, err
		}
		return created, nil
	})
