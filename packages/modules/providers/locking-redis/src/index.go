// Package lockingredis mirrors @medusajs/locking-redis.
package lockingredis

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/locking-redis/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/providers/locking-redis/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return loaders.Load(opts)
	},
}
