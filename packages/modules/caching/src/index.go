// Package caching mirrors @medusajs/caching: the caching module (feature
// flag "caching"), its built in memory provider and the default strategy
// (keys hashed from the input, tags computed from the data, invalidation
// by the entity events of the event bus).
package caching

import (
	"context"

	coremodulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/services"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/types"
)

// service is the caching module service with the container it resolves the
// event bus from.
type service struct {
	*services.CachingModuleService
	container modulessdk.Resolver
}

// Module is the module definition.
var Module = modulessdk.ModuleExports{
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		o := types.OptionsFromMap(opts.Options)
		registry, defaultProvider, err := loaders.LoadProviders(opts.Providers, o, opts.Logger)
		if err != nil {
			return nil, err
		}
		return &service{CachingModuleService: services.NewCachingModuleService(registry, defaultProvider, loaders.Hasher, o.TTL, opts.Logger), container: opts.Container}, nil
	},
	// __hooks.onApplicationStart: the strategy gets the merged schema of the
	// loaded modules and subscribes to the event bus.
	OnApplicationStart: func(_ context.Context, svc any) error {
		s, ok := svc.(*service)
		if !ok {
			return nil
		}
		var bus any
		if s.container != nil {
			bus = s.container.Resolve(utilsmodulessdk.EVENT_BUS)
		}
		return s.CachingModuleService.OnApplicationStart(coremodulessdk.GetAllJoinerConfigs(), bus)
	},
}
