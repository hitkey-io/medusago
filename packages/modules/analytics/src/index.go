// Package analytics mirrors @medusajs/analytics.
package analytics

import (
	analyticstypes "github.com/hitkey-io/medusago/packages/core/types/src/analytics"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/analytics/src/services"
)

// Module is the module definition: the service uses the configured
// analytics provider (exactly one).
var Module = modulessdk.ModuleExports{
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		var providers []analyticstypes.IAnalyticsProvider
		for _, p := range opts.Providers {
			if inst, ok := p.Instance.(analyticstypes.IAnalyticsProvider); ok {
				providers = append(providers, inst)
			}
		}
		provider, err := services.NewAnalyticsProviderService(providers)
		if err != nil {
			return nil, err
		}
		return services.NewAnalyticsService(provider), nil
	},
}
