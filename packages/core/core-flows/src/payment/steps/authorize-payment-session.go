// Package steps mirrors @medusajs/core-flows/src/payment/steps.
package steps

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type paymentService interface {
	Internal(className string) *utilsmodulessdk.InternalService
	AuthorizePaymentSession(ctx context.Context, id string, context_ *dal.Record) (*dal.Record, error)
	CancelPayment(ctx context.Context, id any) (*dal.Record, error)
}

// AuthorizePaymentSessionStepInput mirrors AuthorizePaymentSessionStepInput.
type AuthorizePaymentSessionStepInput struct {
	ID      string
	Context *dal.Record
}

// AuthorizePaymentSessionStep mirrors authorizePaymentSessionStep.
var AuthorizePaymentSessionStep = composer.CreateStep("authorize-payment-session-step",
	func(ctx composer.StepExecutionContext, in AuthorizePaymentSessionStepInput) (composer.StepResponse[*dal.Record], error) {
		svc := ctx.Container.Resolve(utilsmodulessdk.PAYMENT).(paymentService)
		if in.ID == "" {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		c := in.Context
		if c == nil {
			c = dal.NewRecord()
		}
		payment, err := svc.AuthorizePaymentSession(ctx.Context, in.ID, c)
		if err != nil {
			slog.Error("Error was thrown trying to authorize payment session - " + in.ID + " - " + err.Error())
			if _, ok := err.(*common.MedusaError); ok {
				return composer.StepResponse[*dal.Record]{}, err
			}
		}
		session, err := svc.Internal("PaymentSession").Retrieve(ctx.Context, in.ID, utilsmodulessdk.FindConfig{Relations: []string{"payment", "payment.captures"}})
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		status, _ := session.Get("status")
		if status == "pending_authorization" {
			return composer.NewStepResponse[*dal.Record](nil), nil
		}
		if status == "requires_more" {
			return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.PAYMENT_REQUIRES_MORE_ERROR, "More information is required for payment")
		}
		if status != "authorized" || payment == nil {
			return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.PAYMENT_AUTHORIZATION_ERROR, "Payment authorization failed")
		}
		p, _ := session.Get("payment")
		rec, _ := p.(*dal.Record)
		return composer.NewStepResponse(rec, any(rec)), nil
	},
	// If payment or any other part of complete cart fails post payment step,
	// the payment is canceled.
	func(ctx composer.StepExecutionContext, data any) error {
		payment, _ := data.(*dal.Record)
		if payment == nil {
			return nil
		}
		// A session requiring more is not reverted (the payment is loaded
		// without its session, so this never applies).
		if ps, _ := payment.Get("payment_session"); ps != nil {
			if r, ok := ps.(*dal.Record); ok {
				if st, _ := r.Get("status"); st == "requires_more" {
					return nil
				}
			}
		}
		svc := ctx.Container.Resolve(utilsmodulessdk.PAYMENT).(paymentService)
		id, _ := payment.Get("id")
		if _, err := svc.CancelPayment(ctx.Context, id); err != nil {
			slog.Error(fmt.Sprintf("Error was thrown trying to cancel payment - %v - %s", id, err.Error()))
		}
		return nil
	})
