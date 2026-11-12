package hooks

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/carts"
)

// registerBeforePaymentCollectionRefresh mirrors
// before-payment-collection-refresh.ts: the refreshCartItemsWorkflow
// "beforeRefreshingPaymentCollection" hook refreshes the gift card credit
// lines of the cart (refreshCartGiftCardsWorkflow, run in the transaction
// and event group of the refresh); the compensation cancels that run.
func registerBeforePaymentCollectionRefresh() {
	cartworkflows.RefreshCartItemsWorkflow.RegisterHookWhen("beforeRefreshingPaymentCollection", pluginEnabled,
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			var cartID string
			switch in := gcGet(gcAsRecord(data), "input").(type) {
			case cartworkflows.RefreshCartItemsInput:
				cartID = in.CartID
			case *cartworkflows.RefreshCartItemsInput:
				cartID = in.CartID
			case *dal.Record:
				cartID, _ = gcGet(in, "cart_id").(string)
			}
			// the shared step context carries preventReleaseEvents: true when
			// the refresh runs as a step of another workflow
			result, err := carts.RefreshCartGiftCardsWorkflow.Run(ctx.Context, ctx.Container,
				carts.RefreshCartGiftCardsWorkflowInput{CartID: cartID},
				composer.RunOptions{TransactionID: ctx.TransactionID, EventGroupID: ctx.EventGroupID, PreventReleaseEvents: ctx.IsSubWorkflow()})
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
			_, err := composer.Cancel(ctx.Context, ctx.Container, carts.RefreshCartGiftCardsWorkflow.ID, composer.CancelOptions{TransactionID: transactionID})
			return err
		})
}

func gcAsRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}
