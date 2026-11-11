package services

import (
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// BlikProviderServiceIdentifier mirrors BlikProviderService.identifier.
const BlikProviderServiceIdentifier = types.Blik

// NewBlikProviderService mirrors BlikProviderService.
func NewBlikProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(types.StripeOptions) types.PaymentIntentOptions {
		automatic := "automatic"
		return types.PaymentIntentOptions{PaymentMethodTypes: []string{"blik"}, CaptureMethod: &automatic}
	})
}
