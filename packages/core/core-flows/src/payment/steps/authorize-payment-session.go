// Package steps mirrors @medusajs/core-flows/src/payment/steps.
package steps

import (
	"context"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type paymentService interface {
	Internal(className string) *utilsmodulessdk.InternalService
	AuthorizePaymentSession(ctx context.Context, id string, context_ *dal.Record) (*dal.Record, error)
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
	func(ctx composer.StepExecutionContext, data any) error {
		payment, _ := data.(*dal.Record)
		if payment == nil {
			return nil
		}
		slog.Error("cancelling a payment is not supported yet")
		return nil
	})
