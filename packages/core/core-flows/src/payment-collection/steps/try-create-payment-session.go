package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// TryCreatePaymentSessionStepID mirrors tryCreatePaymentSessionStepId.
const TryCreatePaymentSessionStepID = "try-create-payment-session"

// TryCreatePaymentSessionStep mirrors tryCreatePaymentSessionStep: it
// creates a payment session like CreatePaymentSessionStep, but reports an
// account holder that the payment provider no longer has instead of failing.
// The output is { session, account_holder_not_found: false } or
// { account_holder_not_found: true }.
var TryCreatePaymentSessionStep = composer.CreateStep(TryCreatePaymentSessionStepID,
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[*dal.Record], error) {
		p := dal.NewRecord()
		p.Set("provider_id", g(in, "provider_id"))
		p.Set("currency_code", g(in, "currency_code"))
		p.Set("amount", g(in, "amount"))
		data := g(in, "data")
		if r, isRec := data.(*dal.Record); data == nil || (isRec && r == nil) {
			data = dal.NewRecord()
		}
		p.Set("data", data)
		p.Set("context", g(in, "context"))
		meta := g(in, "metadata")
		if r, isRec := meta.(*dal.Record); meta == nil || (isRec && r == nil) {
			meta = dal.NewRecord()
		}
		p.Set("metadata", meta)
		id, _ := g(in, "payment_collection_id").(string)
		session, err := PaymentOf(ctx.Container).CreatePaymentSession(ctx.Context, id, p)
		if err != nil {
			ctxRec, _ := g(in, "context").(*dal.Record)
			if me, ok := common.AsMedusaError(err); ok && isTruthy(g(ctxRec, "account_holder")) &&
				me.Code == common.MedusaErrorCodes.ACCOUNT_HOLDER_NOT_FOUND {
				out := dal.NewRecord()
				out.Set("account_holder_not_found", true)
				// no compensation input: the compensation receives the output
				return composer.NewStepResponse(out), nil
			}
			return composer.StepResponse[*dal.Record]{}, err
		}
		out := dal.NewRecord()
		out.Set("session", session)
		out.Set("account_holder_not_found", false)
		return composer.NewStepResponse(out, g(session, "id")), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		if !isTruthy(data) {
			return nil
		}
		return PaymentOf(ctx.Container).DeletePaymentSession(ctx.Context, data)
	})

// isTruthy mirrors a JS truthiness check of a step value.
func isTruthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case *dal.Record:
		return t != nil
	case string:
		return t != ""
	case bool:
		return t
	}
	return true
}
