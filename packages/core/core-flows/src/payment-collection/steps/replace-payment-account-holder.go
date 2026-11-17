package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ReplacePaymentAccountHolderStepID mirrors replacePaymentAccountHolderStepId.
const ReplacePaymentAccountHolderStepID = "replace-payment-account-holder"

// ReplacePaymentAccountHolderStep mirrors replacePaymentAccountHolderStep:
// it replaces an account holder's record in the payment provider with a new
// one (input: { id, provider_id, context: { customer } }).
var ReplacePaymentAccountHolderStep = composer.CreateStep(ReplacePaymentAccountHolderStepID,
	func(ctx composer.StepExecutionContext, in *dal.Record) (composer.StepResponse[*dal.Record], error) {
		holder, err := PaymentOf(ctx.Container).ReplaceAccountHolder(ctx.Context, in)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(holder), nil
	}, nil)
