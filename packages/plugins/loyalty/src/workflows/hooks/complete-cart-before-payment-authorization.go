package hooks

import (
	"fmt"

	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/carts"
)

// registerCompleteCartBeforePaymentAuthorization mirrors
// complete-cart-before-payment-authorization.ts: the completeCartWorkflow
// "beforePaymentAuthorization" hook debits the store credit and gift card
// accounts of the cart's credit lines (confirmCartCreditLinesWorkflow, run in
// the transaction and event group of the cart completion, its events left
// to it); the compensation cancels that run.
func registerCompleteCartBeforePaymentAuthorization() {
	cartworkflows.CompleteCartWorkflow.RegisterHookWhen("beforePaymentAuthorization", pluginEnabled,
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			rec, _ := data.(*dal.Record)
			var input *dal.Record
			if rec != nil {
				v, _ := rec.Get("input")
				input, _ = v.(*dal.Record)
			}
			if input == nil {
				return composer.StepResponse[any]{}, fmt.Errorf("Cannot read properties of undefined (reading 'id')")
			}
			id, _ := input.Get("id")
			cartID, _ := id.(string)
			result, err := carts.ConfirmCartCreditLinesWorkflow.Run(ctx.Context, ctx.Container,
				carts.ConfirmCartCreditLinesWorkflowInput{CartID: cartID},
				composer.RunOptions{TransactionID: ctx.TransactionID, EventGroupID: ctx.EventGroupID, PreventReleaseEvents: true})
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](result, any(ctx.TransactionID)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			transactionID, _ := data.(string)
			if transactionID == "" {
				return nil
			}
			_, err := composer.Cancel(ctx.Context, ctx.Container, carts.ConfirmCartCreditLinesWorkflow.ID, composer.CancelOptions{TransactionID: transactionID})
			return err
		})
}
