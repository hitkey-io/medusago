package hooks

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/orders"
)

// registerAfterOrderCreated mirrors after-order-created.ts: the
// completeCartWorkflow "orderCreated" hook links the gift cards of the cart
// to the order (cloneCartGiftCardsToOrderWorkflow, run in the transaction
// and event group of the cart completion, its events left to it); the
// compensation cancels that run.
func registerAfterOrderCreated() {
	cartworkflows.CompleteCartWorkflow.RegisterHookWhen("orderCreated", pluginEnabled,
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			rec, _ := data.(*dal.Record)
			orderID, _ := gcGet(rec, "order_id").(string)
			cartID, _ := gcGet(rec, "cart_id").(string)
			result, err := orders.CloneCartGiftCardsToOrderWorkflow.Run(ctx.Context, ctx.Container,
				orders.CloneCartGiftCardsToOrderWorkflowInput{OrderID: orderID, CartID: cartID},
				composer.RunOptions{TransactionID: ctx.TransactionID, EventGroupID: ctx.EventGroupID, PreventReleaseEvents: true})
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse(result, any(ctx.TransactionID)), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			transactionID, _ := data.(string)
			if transactionID == "" {
				return nil
			}
			_, err := composer.Cancel(ctx.Context, ctx.Container, orders.CloneCartGiftCardsToOrderWorkflow.ID, composer.CancelOptions{TransactionID: transactionID})
			return err
		})
}

// gcGet returns r[k] (nil when absent or r is nil).
func gcGet(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}
