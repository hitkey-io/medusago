// Package workflows mirrors @medusajs/core-flows/src/payment/workflows.
package workflows

import (
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	orderpayments "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows/payments"
	paymentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/defaults"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func g(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func recs(v any) []*dal.Record {
	switch l := v.(type) {
	case []*dal.Record:
		return l
	case []any:
		var out []*dal.Record
		for _, x := range l {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func currencyEpsilon(code any) *totals.BN {
	s, _ := code.(string)
	upper := strings.ToUpper(s)
	for _, c := range defaults.DefaultCurrencies {
		if c.Code == upper {
			return totals.GetEpsilonFromDecimalPrecision(c.DecimalDigits)
		}
	}
	return totals.GetEpsilonFromDecimalPrecision(-1)
}

func sum(list []*dal.Record, key string) *totals.BN {
	out := totals.Convert(0)
	for _, r := range list {
		out = totals.Add(out, g(r, key))
	}
	return out
}

func remote(e *composer.Execution, name, entry string, fields []string, variables *dal.Record, throw bool) (*dal.Record, error) {
	list := false
	step := commonsteps.UseRemoteQueryStep
	if name != "" {
		step = step.Config(name, composer.StepOptions{})
	}
	res, err := step.Run(e, commonsteps.RemoteStepInput{EntryPoint: entry, Fields: fields, Variables: variables, List: &list, ThrowIfKeyNotFound: throw})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	r, _ := res.Data[0].(*dal.Record)
	return r, nil
}

func kv(k string, v any) *dal.Record {
	r := dal.NewRecord()
	r.Set(k, v)
	return r
}

// CapturePaymentWorkflow mirrors capturePaymentWorkflow.
var CapturePaymentWorkflow = composer.CreateWorkflow("capture-payment-workflow",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		payment, err := paymentsteps.CapturePaymentStep.Run(e, input)
		if err != nil {
			return nil, err
		}
		orderPayment, err := remote(e, "", "order_payment_collection", []string{"order.id"}, kv("payment_collection_id", g(payment, "payment_collection_id")), false)
		if err != nil {
			return nil, err
		}
		if orderID := g(asRec(g(orderPayment, "order")), "id"); orderID != nil {
			var tx []*dal.Record
			for _, c := range recs(g(payment, "captures")) {
				t := dal.NewRecord()
				t.Set("order_id", orderID)
				amount := g(input, "amount")
				if amount == nil {
					amount = g(c, "raw_amount")
					if amount == nil {
						amount = g(c, "amount")
					}
				}
				t.Set("amount", amount)
				t.Set("currency_code", g(payment, "currency_code"))
				t.Set("reference_id", g(c, "id"))
				t.Set("reference", "capture")
				tx = append(tx, t)
			}
			if _, err := ordersteps.AddOrderTransactionStep.Run(e, tx); err != nil {
				return nil, err
			}
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "payment.captured", Data: kv("id", g(payment, "id"))})
		return payment, err
	})

func asRec(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

var validateRefundPaymentExceedsCapturedAmountStep = composer.CreateStep("validate-refund-payment-exceeds-captured-amount",
	func(ctx composer.StepExecutionContext, in struct {
		Payment      *dal.Record
		RefundAmount any
	}) (composer.StepResponse[any], error) {
		captured := sum(recs(g(in.Payment, "captures")), "raw_amount")
		refunded := sum(recs(g(in.Payment, "refunds")), "raw_amount")
		total := totals.Add(refunded, in.RefundAmount)
		if totals.Lt(totals.Sub(captured, total), totals.Mult(currencyEpsilon(g(in.Payment, "currency_code")), -1)) {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "You are not allowed to refund more than the captured amount")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RefundPaymentWorkflow mirrors refundPaymentWorkflow.
var RefundPaymentWorkflow = composer.CreateWorkflow("refund-payment-workflow",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		payment, err := remote(e, "", "payment", []string{"id", "payment_collection_id", "currency_code", "amount", "raw_amount", "captures.raw_amount", "refunds.raw_amount"},
			kv("id", g(input, "payment_id")), true)
		if err != nil {
			return nil, err
		}
		// new WorkflowResponse(payment)
		composer.SetFailedResult(e, payment)
		if amount := g(input, "amount"); amount != nil && !totals.Eq(amount, 0) {
			if _, err := validateRefundPaymentExceedsCapturedAmountStep.Run(e, struct {
				Payment      *dal.Record
				RefundAmount any
			}{payment, amount}); err != nil {
				return nil, err
			}
		}
		opc, err := remote(e, "order-payment-collection", "order_payment_collection", []string{"order.id"}, kv("payment_collection_id", g(payment, "payment_collection_id")), true)
		if err != nil {
			return nil, err
		}
		orderID := g(asRec(g(opc, "order")), "id")
		order, err := remote(e, "order", "order", []string{"id", "summary", "total", "currency_code", "region_id"}, kv("id", orderID), true)
		if err != nil {
			return nil, err
		}
		var refundReason *dal.Record
		if g(input, "refund_reason_id") != nil {
			refundReason, err = remote(e, "refund-reason", "refund_reason", []string{"id", "label", "code"}, kv("id", g(input, "refund_reason_id")), true)
			if err != nil {
				return nil, err
			}
		}
		// a replay of an already-succeeded refund moves no funds
		key := g(input, "idempotency_key")
		isList := false
		existingRefund, err := composer.When(e, "fetch-refund-by-idempotency-key", key != nil && key != "", func() (*dal.Record, error) {
			f := dal.NewRecord()
			f.Set("payment_id", g(input, "payment_id"))
			f.Set("idempotency_key", key)
			r, err := commonsteps.UseQueryGraphStep.Config("refund-by-idempotency-key", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
				GraphInput: query.GraphInput{Entity: "refund", Fields: []string{"id"}, Filters: f}, IsList: &isList})
			if err != nil {
				return nil, err
			}
			return asRec(r.Data), nil
		})
		if err != nil {
			return nil, err
		}
		refundTransaction, err := composer.When(e, "fetch-refund-order-transaction", existingRefund != nil && g(existingRefund, "id") != nil, func() (*dal.Record, error) {
			tf := dal.NewRecord()
			tf.Set("reference", "refund")
			tf.Set("reference_id", g(existingRefund, "id"))
			t, err := commonsteps.UseQueryGraphStep.Config("order-transaction-by-refund", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
				GraphInput: query.GraphInput{Entity: "order_transaction", Fields: []string{"id"}, Filters: tf}, IsList: &isList})
			if err != nil {
				return nil, err
			}
			return asRec(t.Data), nil
		})
		if err != nil {
			return nil, err
		}
		isReplay := refundTransaction != nil && g(refundTransaction, "id") != nil
		refundPayment, err := paymentsteps.RefundPaymentStep.Run(e, input)
		if err != nil {
			return nil, err
		}
		summary := asRec(g(order, "summary"))
		pending := g(summary, "raw_pending_difference")
		if pending == nil {
			pending = g(summary, "pending_difference")
		}
		if pending == nil {
			pending = 0
		}
		amountToRefund := g(input, "amount")
		if amountToRefund == nil {
			amountToRefund = g(payment, "raw_amount")
			if amountToRefund == nil {
				amountToRefund = g(payment, "amount")
			}
		}
		var creditLineAmount any = amountToRefund
		if totals.Lt(pending, 0) {
			owed := totals.Mult(pending, -1)
			if totals.Gt(amountToRefund, owed) {
				creditLineAmount = totals.Sub(amountToRefund, owed)
			} else {
				creditLineAmount = 0
			}
		}
		if orderID != nil {
			var tx []*dal.Record
			for _, r := range recs(g(refundPayment, "refunds")) {
				amount := g(input, "amount")
				if amount == nil {
					amount = g(r, "raw_amount")
					if amount == nil {
						amount = g(r, "amount")
					}
				}
				t := dal.NewRecord()
				t.Set("order_id", orderID)
				t.Set("amount", totals.Mult(amount, -1))
				cc := g(refundPayment, "currency_code")
				if cc == nil {
					cc = g(order, "currency_code")
				}
				t.Set("currency_code", cc)
				t.Set("reference_id", g(r, "id"))
				t.Set("reference", "refund")
				tx = append(tx, t)
			}
			if _, err := ordersteps.AddOrderTransactionStep.Run(e, tx); err != nil {
				return nil, err
			}
		}
		if !isReplay && totals.Gt(creditLineAmount, 0) {
			if _, err := orderpayments.CreateOrderRefundCreditLinesWorkflow.RunAsStep(e, orderpayments.CreateOrderRefundCreditLinesInput{
				OrderID: g(order, "id"), Amount: creditLineAmount, Reference: g(refundReason, "label"), ReferenceID: g(refundReason, "code"),
			}); err != nil {
				return nil, err
			}
		}
		if !isReplay {
			if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "payment.refunded", Data: kv("id", g(payment, "id"))}); err != nil {
				return nil, err
			}
		}
		return payment, nil
	})

var validatePaymentsRefundStep = composer.CreateStep("validate-payments-refund-step",
	func(ctx composer.StepExecutionContext, in struct {
		Payments []*dal.Record
		Input    []*dal.Record
	}) (composer.StepResponse[any], error) {
		amounts := map[any]any{}
		for _, i := range in.Input {
			amounts[g(i, "payment_id")] = g(i, "amount")
		}
		for _, p := range in.Payments {
			captured := sum(recs(g(p, "captures")), "raw_amount")
			refunded := sum(recs(g(p, "refunds")), "raw_amount")
			total := totals.Add(refunded, amounts[g(p, "id")])
			if totals.Lt(totals.Sub(captured, total), totals.Mult(currencyEpsilon(g(p, "currency_code")), -1)) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA,
					fmt.Sprintf("Payment with id %v is trying to refund amount greater than the refundable amount", g(p, "id")))
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RefundPaymentsWorkflow mirrors refundPaymentsWorkflow.
var RefundPaymentsWorkflow = composer.CreateWorkflow("refund-payments-workflow",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		var ids []any
		for _, i := range input {
			ids = append(ids, g(i, "payment_id"))
		}
		res, err := commonsteps.UseQueryGraphStep.Config("get-cart", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "payments", Fields: []string{"id", "currency_code", "refunds.id", "refunds.amount", "refunds.raw_amount", "captures.id",
				"captures.amount", "captures.raw_amount", "payment_collection.order.id", "payment_collection.order.currency_code"}, Filters: kv("id", ids)},
			Options: joiner.Options{ThrowIfKeyNotFound: true},
		})
		if err != nil {
			return nil, err
		}
		payments := recs(res.Data)
		if _, err := validatePaymentsRefundStep.Run(e, struct {
			Payments []*dal.Record
			Input    []*dal.Record
		}{payments, input}); err != nil {
			return nil, err
		}
		refunded, err := paymentsteps.RefundPaymentsStep.Run(e, input)
		if err != nil {
			return nil, err
		}
		byID := map[any]*dal.Record{}
		for _, p := range payments {
			byID[g(p, "id")] = p
		}
		var tx []*dal.Record
		for _, i := range input {
			p := byID[g(i, "payment_id")]
			order := asRec(g(asRec(g(p, "payment_collection")), "order"))
			if order == nil {
				continue
			}
			t := dal.NewRecord()
			t.Set("order_id", g(order, "id"))
			t.Set("amount", totals.Mult(g(i, "amount"), -1))
			t.Set("currency_code", g(p, "currency_code"))
			t.Set("reference_id", g(p, "id"))
			t.Set("reference", "refund")
			tx = append(tx, t)
		}
		if _, err := ordersteps.AddOrderTransactionStep.Run(e, tx); err != nil {
			return nil, err
		}
		return refunded, nil
	})
