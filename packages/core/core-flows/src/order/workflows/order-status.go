package workflows

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	orderpayments "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows/payments"
	paymentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/steps"
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/workflows"
	reservationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/reservation/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(ids []any) []any {
	out := []any{}
	for _, id := range ids {
		r := dal.NewRecord()
		r.Set("id", id)
		out = append(out, r)
	}
	return out
}

// CompleteOrderWorkflow mirrors completeOrderWorkflow.
var CompleteOrderWorkflow = composer.CreateWorkflow("complete-order-workflow",
	func(e *composer.Execution, in struct {
		OrderIDs       []any
		AdditionalData any
	}) ([]*dal.Record, error) {
		completed, err := ordersteps.CompleteOrdersStep.Run(e, in.OrderIDs)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.completed", Data: idEvents(in.OrderIDs)}); err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("orders", completed)
		hook.Set("additional_data", in.AdditionalData)
		return completed, composer.CreateHook(e, "ordersCompleted", hook)
	})

// ArchiveOrderWorkflow mirrors archiveOrderWorkflow.
var ArchiveOrderWorkflow = composer.CreateWorkflow("archive-order-workflow",
	func(e *composer.Execution, ids []any) ([]*dal.Record, error) {
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.archived", Data: idEvents(ids)}); err != nil {
			return nil, err
		}
		return ordersteps.ArchiveOrdersStep.Run(e, ids)
	})

func orderPayments(order *dal.Record) []*dal.Record {
	var out []*dal.Record
	for _, pc := range recs(g(order, "payment_collections")) {
		out = append(out, recs(g(pc, "payments"))...)
	}
	return out
}

// RefundCapturedPaymentsWorkflow mirrors refundCapturedPaymentsWorkflow.
var RefundCapturedPaymentsWorkflow = composer.CreateWorkflow("refund-captured-payments-workflow",
	func(e *composer.Execution, in struct {
		OrderID   any
		CreatedBy any
		Note      any
	}) ([]*dal.Record, error) {
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "orders", Fields: []string{"id", "status", "summary", "total", "payment_collections.payments.id", "payment_collections.payments.amount",
				"payment_collections.payments.refunds.id", "payment_collections.payments.refunds.amount", "payment_collections.payments.captures.id",
				"payment_collections.payments.captures.amount"}, Filters: f},
			Options: joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var order *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			order = list[0]
		}
		var data []*dal.Record
		total := totals.Convert(0)
		for _, p := range orderPayments(order) {
			if len(recs(g(p, "captures"))) == 0 {
				continue
			}
			captured := totals.Convert(0)
			for _, c := range recs(g(p, "captures")) {
				captured = totals.Sum(captured, g(c, "amount"))
			}
			refunded := totals.Convert(0)
			for _, r := range recs(g(p, "refunds")) {
				refunded = totals.Sum(refunded, g(r, "amount"))
			}
			amount := totals.Sub(captured, refunded)
			if !totals.Gt(amount, 0) {
				continue
			}
			r := dal.NewRecord()
			r.Set("payment_id", g(p, "id"))
			if in.CreatedBy != nil {
				r.Set("created_by", in.CreatedBy)
			}
			r.Set("amount", amount)
			if in.Note != nil {
				r.Set("note", in.Note)
			}
			data = append(data, r)
			total = totals.Sum(total, amount)
		}
		if !totals.Gt(total, 0) {
			return []*dal.Record{}, nil
		}
		return paymentworkflows.RefundPaymentsWorkflow.RunAsStep(e, data)
	})

var cancelValidateOrder = composer.CreateStep("cancel-validate-order",
	func(ctx composer.StepExecutionContext, order *dal.Record) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if g(order, "status") == "completed" {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Cannot cancel a completed order. Please use the return process to handle refunds or exchanges.")
		}
		for _, f := range recs(g(order, "fulfillments")) {
			if g(f, "canceled_at") == nil {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "All fulfillments must be canceled before canceling an order")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CancelOrderWorkflow mirrors cancelOrderWorkflow.
var CancelOrderWorkflow = composer.CreateWorkflow("cancel-order",
	func(e *composer.Execution, in struct {
		OrderID    any
		CanceledBy any
	}) (any, error) {
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		res, err := commonsteps.UseQueryGraphStep.Config("get-order", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: []string{"id", "status", "items.id", "fulfillments.canceled_at", "payment_collections.payments.id",
				"payment_collections.payments.amount", "payment_collections.payments.refunds.id", "payment_collections.payments.refunds.amount",
				"payment_collections.payments.refunds.raw_amount", "payment_collections.payments.captures.id", "payment_collections.payments.captures.amount",
				"payment_collections.payments.captures.raw_amount"}, Filters: f},
			Options: joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		var order *dal.Record
		if list := recs(res.Data); len(list) > 0 {
			order = list[0]
		}
		if _, err := cancelValidateOrder.Run(e, order); err != nil {
			return nil, err
		}
		payments := orderPayments(order)
		var uncaptured []any
		for _, p := range payments {
			if len(recs(g(p, "captures"))) == 0 {
				uncaptured = append(uncaptured, g(p, "id"))
			}
		}
		var lineItemIDs []any
		for _, i := range recs(g(order, "items")) {
			lineItemIDs = append(lineItemIDs, g(i, "id"))
		}
		var refundedPayments []*dal.Record
		var deletedReservationIDs []any
		// parallelize(refundCapturedPayments, deleteReservations,
		// cancelPayment, emitEvent): the branches run in the order upstream
		// completes them (the event step, the payment cancellation, the
		// reservations, then the refund workflow)
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.canceled", Data: idEvent(g(order, "id"))})
				return err
			},
			func() error { _, err := paymentsteps.CancelPaymentStep.Run(e, uncaptured); return err },
			func() error {
				var err error
				deletedReservationIDs, err = reservationsteps.DeleteReservationsByLineItemsStep.Run(e, lineItemIDs)
				return err
			},
			func() error {
				var err error
				refundedPayments, err = RefundCapturedPaymentsWorkflow.RunAsStep(e, struct {
					OrderID   any
					CreatedBy any
					Note      any
				}{OrderID: g(order, "id"), CreatedBy: in.CanceledBy})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		events := []any{}
		for _, id := range deletedReservationIDs {
			r := dal.NewRecord()
			r.Set("id", id)
			r.Set("order_id", g(order, "id"))
			events = append(events, r)
		}
		if _, err := commonsteps.EmitEventStep.Config("emit-reservation-item-deleted", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.deleted", Data: events}); err != nil {
			return nil, err
		}
		var refundedIDs []any
		refundedSet := map[any]bool{}
		for _, p := range refundedPayments {
			refundedIDs = append(refundedIDs, g(p, "id"))
			refundedSet[g(p, "id")] = true
		}
		rf := dal.NewRecord()
		if refundedIDs == nil {
			refundedIDs = []any{}
		}
		rf.Set("payment_id", refundedIDs)
		refunds, err := commonsteps.UseQueryGraphStep.Config("get-refunded-payment-refunds", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "refunds", Fields: []string{"raw_amount"}, Filters: rf},
		})
		if err != nil {
			return nil, err
		}
		after := totals.Convert(0)
		for _, r := range recs(refunds.Data) {
			after = totals.Sum(after, g(r, "raw_amount"))
		}
		before := totals.Convert(0)
		for _, p := range payments {
			if !refundedSet[g(p, "id")] {
				continue
			}
			for _, r := range recs(g(p, "refunds")) {
				amount := g(r, "raw_amount")
				if amount == nil {
					amount = g(r, "amount")
				}
				before = totals.Sum(before, amount)
			}
		}
		if _, err := orderpayments.CreateOrderRefundCreditLinesWorkflow.RunAsStep(e, orderpayments.CreateOrderRefundCreditLinesInput{OrderID: g(order, "id"), Amount: totals.Sub(after, before)}); err != nil {
			return nil, err
		}
		var pcIDs []any
		for _, pc := range recs(g(order, "payment_collections")) {
			pcIDs = append(pcIDs, g(pc, "id"))
		}
		if len(pcIDs) > 0 {
			sel := dal.NewRecord()
			sel.Set("id", pcIDs)
			upd := dal.NewRecord()
			upd.Set("status", "canceled")
			u := dal.NewRecord()
			u.Set("selector", sel)
			u.Set("update", upd)
			if _, err := cartworkflows.UpdatePaymentCollectionStep.Run(e, u); err != nil {
				return nil, err
			}
		}
		if _, err := ordersteps.CancelOrdersStep.Run(e, []any{g(order, "id")}); err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("order", order)
		return nil, composer.CreateHook(e, "orderCanceled", hook)
	})

func idEvent(id any) *dal.Record {
	r := dal.NewRecord()
	r.Set("id", id)
	return r
}
