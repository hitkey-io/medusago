// Package authgoogle mirrors @medusajs/auth-google.
package authgoogle

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/auth-google/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return services.NewGoogleAuthService(opts.Options, opts.Logger)
	},
}
