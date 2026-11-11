package services

import (
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// PromptpayProviderServiceIdentifier mirrors PromptpayProviderService.identifier.
const PromptpayProviderServiceIdentifier = types.PromptPay

// NewPromptpayProviderService mirrors PromptpayProviderService.
func NewPromptpayProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(types.StripeOptions) types.PaymentIntentOptions {
		automatic := "automatic"
		return types.PaymentIntentOptions{PaymentMethodTypes: []string{"promptpay"}, CaptureMethod: &automatic}
	})
}
