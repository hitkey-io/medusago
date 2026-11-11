// Package paymentstripe mirrors @medusajs/payment-stripe.
package paymentstripe

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/core"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-stripe/src/services"
)

func service(identifier string, build func(options map[string]any) *core.StripeBase) modulessdk.ProviderService {
	return modulessdk.ProviderService{
		Identifier:      identifier,
		ValidateOptions: core.ValidateOptions,
		NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
			return build(opts.Options), nil
		},
	}
}

// Module is the module provider definition (ModuleProvider(Modules.PAYMENT,
// { services })), the services in upstream order.
var Module = modulessdk.ModuleExports{
	ProviderServices: []modulessdk.ProviderService{
		service(services.BancontactProviderServiceIdentifier, services.NewBancontactProviderService),
		service(services.BlikProviderServiceIdentifier, services.NewBlikProviderService),
		service(services.GiropayProviderServiceIdentifier, services.NewGiropayProviderService),
		service(services.IdealProviderServiceIdentifier, services.NewIdealProviderService),
		service(services.StripeProviderServiceIdentifier, services.NewStripeProviderService),
		service(services.Przelewy24ProviderServiceIdentifier, services.NewPrzelewy24ProviderService),
		service(services.PromptpayProviderServiceIdentifier, services.NewPromptpayProviderService),
		service(services.OxxoProviderServiceIdentifier, services.NewOxxoProviderService),
	},
}
