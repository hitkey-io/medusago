// Package authgithub mirrors @medusajs/auth-github.
package authgithub

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/auth-github/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return services.NewGithubAuthService(opts.Options, opts.Logger)
	},
}
