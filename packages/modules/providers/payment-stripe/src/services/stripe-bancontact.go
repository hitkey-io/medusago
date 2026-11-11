package services

import (
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// BancontactProviderServiceIdentifier mirrors BancontactProviderService.identifier.
const BancontactProviderServiceIdentifier = types.BanContact

// NewBancontactProviderService mirrors BancontactProviderService.
func NewBancontactProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(types.StripeOptions) types.PaymentIntentOptions {
		automatic := "automatic"
		return types.PaymentIntentOptions{PaymentMethodTypes: []string{"bancontact"}, CaptureMethod: &automatic}
	})
}
