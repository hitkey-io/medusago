package services

import (
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// Przelewy24ProviderServiceIdentifier mirrors Przelewy24ProviderService.identifier.
const Przelewy24ProviderServiceIdentifier = types.Przelewy24

// NewPrzelewy24ProviderService mirrors Przelewy24ProviderService.
func NewPrzelewy24ProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(types.StripeOptions) types.PaymentIntentOptions {
		automatic := "automatic"
		return types.PaymentIntentOptions{PaymentMethodTypes: []string{"p24"}, CaptureMethod: &automatic}
	})
}
