package steps

import (
	"fmt"
	"log/slog"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	paymentcollectionworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/workflows"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// RefundPaymentAndRecreatePaymentSessionInput mirrors
// refundPaymentAndRecreatePaymentSessionWorkflowInput.
type RefundPaymentAndRecreatePaymentSessionInput struct {
	PaymentCollectionID string      `json:"payment_collection_id"`
	ProviderID          string      `json:"provider_id"`
	CustomerID          string      `json:"customer_id"`
	Data                *dal.Record `json:"data"`
	Context             *dal.Record `json:"context"`
	PaymentID           string      `json:"payment_id"`
	Amount              any         `json:"amount"`
	Note                string      `json:"note"`
}

// RefundPaymentAndRecreatePaymentSessionWorkflow mirrors
// refundPaymentAndRecreatePaymentSessionWorkflow (cart/workflows upstream;
// here next to its only caller, the cart workflows import this package).
var RefundPaymentAndRecreatePaymentSessionWorkflow = composer.CreateWorkflow("refund-payment-and-recreate-payment-session",
	func(e *composer.Execution, in RefundPaymentAndRecreatePaymentSessionInput) (*dal.Record, error) {
		refund := dal.NewRecord()
		refund.Set("payment_id", in.PaymentID)
		if in.Note != "" {
			refund.Set("note", in.Note)
		}
		refund.Set("amount", in.Amount)
		if _, err := paymentworkflows.RefundPaymentsWorkflow.RunAsStep(e, []*dal.Record{refund}); err != nil {
			return nil, err
		}
		return paymentcollectionworkflows.CreatePaymentSessionsWorkflow.RunAsStep(e, paymentcollectionworkflows.CreatePaymentSessionsInput{
			PaymentCollectionID: in.PaymentCollectionID,
			ProviderID:          in.ProviderID,
			CustomerID:          in.CustomerID,
			Data:                in.Data,
		})
	})

// CompensatePaymentIfNeededStep mirrors compensatePaymentIfNeededStep: the
// last compensation of the cart completion refunds a captured payment and
// creates a new payment session.
var CompensatePaymentIfNeededStep = composer.CreateStep("compensate-payment-if-needed",
	func(ctx composer.StepExecutionContext, paymentSessionID any) (composer.StepResponse[any], error) {
		return composer.NewStepResponse[any](paymentSessionID, paymentSessionID), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		paymentSessionID, _ := data.(string)
		if paymentSessionID == "" {
			return nil
		}
		q := commonsteps.QueryOf(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", paymentSessionID)
		res, err := q.Graph(ctx.Context, query.GraphInput{Entity: "payment_session", Fields: []string{
			"id", "payment_collection_id", "amount", "raw_amount", "provider_id", "data",
			"payment.id", "payment.captured_at", "payment.customer.id",
		}, Filters: f}, joiner.Options{})
		if err != nil {
			return err
		}
		if len(res.Data) == 0 {
			return nil
		}
		session, _ := res.Data[0].(*dal.Record)
		payment := Rec(session, "payment")
		if !Truthy(Get(payment, "captured_at")) {
			return nil
		}
		lf := dal.NewRecord()
		lf.Set("payment_collection_id", Get(session, "payment_collection_id"))
		links, err := q.Graph(ctx.Context, query.GraphInput{Entity: "cart_payment_collection", Fields: []string{"cart_id"}, Filters: lf}, joiner.Options{})
		if err != nil {
			return err
		}
		var cartID any
		if len(links.Data) > 0 {
			cartID = Get(links.Data[0].(*dal.Record), "cart_id")
		}
		if Truthy(cartID) {
			of := dal.NewRecord()
			of.Set("cart_id", cartID)
			orders, err := q.Graph(ctx.Context, query.GraphInput{Entity: "order_cart", Fields: []string{"order_id"}, Filters: of}, joiner.Options{})
			if err != nil {
				return err
			}
			if len(orders.Data) > 0 {
				if orderID := Get(orders.Data[0].(*dal.Record), "order_id"); Truthy(orderID) {
					slog.Warn(fmt.Sprintf("Skipping refund of payment %v: a later completion attempt already placed order %v for cart %v.", Get(payment, "id"), orderID, cartID))
					return nil
				}
			}
		}
		amount := Get(session, "raw_amount")
		if amount == nil {
			amount = Get(session, "amount")
		}
		customerID, _ := Get(Rec(payment, "customer"), "id").(string)
		collectionID, _ := Get(session, "payment_collection_id").(string)
		providerID, _ := Get(session, "provider_id").(string)
		paymentID, _ := Get(payment, "id").(string)
		sessionData, _ := Get(session, "data").(*dal.Record)
		if _, err := RefundPaymentAndRecreatePaymentSessionWorkflow.Run(ctx.Context, ctx.Container, RefundPaymentAndRecreatePaymentSessionInput{
			PaymentCollectionID: collectionID,
			ProviderID:          providerID,
			CustomerID:          customerID,
			Data:                sessionData,
			Amount:              amount,
			PaymentID:           paymentID,
			Note:                "Refunded due to cart completion failure",
		}); err != nil {
			slog.Error(fmt.Sprintf("Error was thrown trying to refund payment - %v - %v", Get(payment, "id"), err))
		}
		return nil
	})
