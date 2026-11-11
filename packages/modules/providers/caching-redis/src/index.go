// Package cachingredis mirrors @medusajs/caching-redis: the Redis provider
// of the caching module.
package cachingredis

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/providers/caching-redis/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/providers/caching-redis/src/services"
)

// Module is the module provider definition.
var Module = modulessdk.ModuleExports{
	ProviderIdentifier: services.Identifier,
	NewProvider: func(opts modulessdk.ProviderOptions) (any, error) {
		return loaders.Load(opts)
	},
}
