package services

import (
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// IdealProviderServiceIdentifier mirrors IdealProviderService.identifier.
const IdealProviderServiceIdentifier = types.Ideal

// NewIdealProviderService mirrors IdealProviderService.
func NewIdealProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(types.StripeOptions) types.PaymentIntentOptions {
		automatic := "automatic"
		return types.PaymentIntentOptions{PaymentMethodTypes: []string{"ideal"}, CaptureMethod: &automatic}
	})
}
