// Package analyticslocal mirrors @medusajs/analytics-local.
package analyticslocal

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/analytics-local/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return services.NewLocalAnalyticsService(opts.Options, opts.Logger), nil
	},
}
