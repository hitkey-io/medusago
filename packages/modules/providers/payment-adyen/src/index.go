// Package paymentadyen mirrors @medusajs/payment-adyen.
package paymentadyen

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/payment-adyen/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderServices: []modulessdk.ProviderService{{
		Identifier:      services.Identifier,
		ValidateOptions: services.ValidateOptions,
		NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
			return services.NewAdyenProviderService(opts.Options, opts.Logger), nil
		},
	}},
}
