package services

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/types"
)

// OxxoProviderServiceIdentifier mirrors OxxoProviderService.identifier.
const OxxoProviderServiceIdentifier = types.Oxxo

// NewOxxoProviderService mirrors OxxoProviderService (expires after
// oxxoExpiresDays || 3 days).
func NewOxxoProviderService(options map[string]any) *core.StripeBase {
	return core.NewStripeBase(options, func(o types.StripeOptions) types.PaymentIntentOptions {
		automatic := "automatic"
		days := o.OxxoExpiresDays
		if days == 0 {
			days = 3
		}
		oxxo := dal.NewRecord()
		oxxo.Set("expires_after_days", days)
		opts := dal.NewRecord()
		opts.Set("oxxo", oxxo)
		return types.PaymentIntentOptions{PaymentMethodTypes: []string{"oxxo"}, CaptureMethod: &automatic, PaymentMethodOptions: opts}
	})
}
