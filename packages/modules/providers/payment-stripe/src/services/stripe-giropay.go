package services

import (
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// GiropayProviderServiceIdentifier mirrors GiropayProviderService.identifier.
const GiropayProviderServiceIdentifier = types.Giropay

// NewGiropayProviderService mirrors GiropayProviderService.
func NewGiropayProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(types.StripeOptions) types.PaymentIntentOptions {
		automatic := "automatic"
		return types.PaymentIntentOptions{PaymentMethodTypes: []string{"giropay"}, CaptureMethod: &automatic}
	})
}
