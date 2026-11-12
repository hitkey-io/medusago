package hooks

import (
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/orders"
)

// scRecords returns the records of a list.
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

// registerAfterOrderCreditLinesCreated mirrors
// after-order-credit-lines-created.ts: the createOrderCreditLinesWorkflow
// "creditLinesCreated" hook refunds the negative credit lines to the
// customer's store credit account (refundCreditLinesWorkflow, run in the
// transaction and event group of the hook, releasing them when it
// finishes); the compensation cancels that run.
func registerAfterOrderCreditLinesCreated() {
	orderworkflows.CreateOrderCreditLinesWorkflow.RegisterHookWhen("creditLinesCreated", pluginEnabled,
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			rec, _ := data.(*dal.Record)
			var orderID string
			var creditLines []*dal.Record
			if rec != nil {
				v, _ := rec.Get("order_id")
				orderID, _ = v.(string)
				cl, _ := rec.Get("credit_lines")
				creditLines = scRecords(cl)
			}
			result, err := orders.RefundCreditLinesWorkflow.Run(ctx.Context, ctx.Container,
				orders.RefundCreditLinesWorkflowInput{OrderID: orderID, CreditLines: creditLines},
				composer.RunOptions{TransactionID: ctx.TransactionID, EventGroupID: ctx.EventGroupID})
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
			_, err := composer.Cancel(ctx.Context, ctx.Container, orders.RefundCreditLinesWorkflow.ID, composer.CancelOptions{TransactionID: transactionID})
			return err
		})
}
