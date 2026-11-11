// Package analyticsposthog mirrors @medusajs/analytics-posthog.
package analyticsposthog

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/analytics-posthog/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return services.NewPosthogAnalyticsService(opts.Options, opts.Logger)
	},
}
