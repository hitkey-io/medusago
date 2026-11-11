// Package services mirrors @medusajs/payment-stripe/src/services.
package services

import (
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// StripeProviderServiceIdentifier mirrors StripeProviderService.identifier.
const StripeProviderServiceIdentifier = types.Stripe

// NewStripeProviderService mirrors StripeProviderService.
func NewStripeProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(types.StripeOptions) types.PaymentIntentOptions {
		return types.PaymentIntentOptions{}
	})
}
