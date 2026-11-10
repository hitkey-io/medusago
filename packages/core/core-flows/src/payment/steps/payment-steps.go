package steps

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type paymentWrites interface {
	CapturePayment(ctx context.Context, data *dal.Record) (*dal.Record, error)
	RefundPayment(ctx context.Context, data *dal.Record) (*dal.Record, error)
	CancelPayment(ctx context.Context, id any) (*dal.Record, error)
}

func writes(c composer.Container) paymentWrites {
	return c.Resolve(utilsmodulessdk.PAYMENT).(paymentWrites)
}

// CapturePaymentStep mirrors capturePaymentStep.
var CapturePaymentStep = composer.CreateStep("capture-payment-step",
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[*dal.Record], error) {
		p, err := writes(ctx.Container).CapturePayment(ctx.Context, in)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(p), nil
	}, nil)

// RefundPaymentStep mirrors refundPaymentStep.
var RefundPaymentStep = composer.CreateStep("refund-payment-step",
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[*dal.Record], error) {
		p, err := writes(ctx.Container).RefundPayment(ctx.Context, in)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(p), nil
	}, nil)

// RefundPaymentsStep mirrors refundPaymentsStep (failures are logged).
var RefundPaymentsStep = composer.CreateStep("refund-payments-step",
	func(ctx composer.StepExecutionContext, in []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		var out []*dal.Record
		for _, r := range in {
			p, err := writes(ctx.Container).RefundPayment(ctx.Context, r)
			if err != nil {
				id, _ := r.Get("payment_id")
				slog.Error(fmt.Sprintf("Error was thrown trying to cancel payment - %v - %v", id, err))
				continue
			}
			out = append(out, p)
		}
		if out == nil {
			out = []*dal.Record{}
		}
		return composer.NewStepResponse(out), nil
	}, nil)

// CancelPaymentStep mirrors cancelPaymentStep (failures are logged).
var CancelPaymentStep = composer.CreateStep("cancel-payment-step",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		for _, id := range ids {
			if _, err := writes(ctx.Container).CancelPayment(ctx.Context, id); err != nil {
				slog.Error(fmt.Sprintf("Error was thrown trying to cancel payment - %v - %v", id, err))
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)
