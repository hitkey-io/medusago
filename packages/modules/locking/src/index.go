// Package locking mirrors @medusajs/locking: the locking module (default
// in-memory provider, optional locking-postgres / locking-redis providers).
package locking

import (
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/services"
)

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		registry, defaultProvider, err := loaders.LoadProviders(opts.Providers, opts.Logger)
		if err != nil {
			return nil, err
		}
		return services.NewLockingModuleService(registry, defaultProvider), nil
	},
}
