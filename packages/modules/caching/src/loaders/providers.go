package loaders

import (
	"errors"
	"fmt"
	"log/slog"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/providers"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/services"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/types"
)

// LoadProviders mirrors the providers loader: the built in memory provider
// when in_memory.enable is set (default provider "cache-memory"), the
// configured providers, the default one (is_default, or the only one). It
// returns the registry and the default provider id.
func LoadProviders(configured []modulessdk.ProviderInstance, opts types.CachingModuleOptions, logger *slog.Logger) (*services.CacheProviderService, string, error) {
	if logger == nil {
		logger = slog.Default()
	}
	registry := services.NewCacheProviderService(logger)
	defaultProvider := ""
	if opts.InMemoryEnabled {
		registry.Register(providers.MemoryIdentifier, providers.NewMemoryCachingProvider(logger, Hasher, opts.InMemory))
		defaultProvider = providers.MemoryIdentifier
	}
	for _, p := range configured {
		if p.ID == "" {
			return nil, "", fmt.Errorf(`No "id" provided for provider %s`, p.Identifier)
		}
		inst, ok := p.Instance.(types.ICachingProviderService)
		if !ok {
			return nil, "", fmt.Errorf("caching provider %q is not a caching provider", p.ID)
		}
		registry.Register(p.ID, inst)
	}
	isSingleProvider := len(configured) == 1
	hasDefaultProvider := false
	for _, p := range configured {
		if p.IsDefault || isSingleProvider {
			hasDefaultProvider = true
			defaultProvider = p.ID
		}
	}
	if !hasDefaultProvider {
		if !opts.InMemoryEnabled {
			return nil, "", errors.New("[caching-module]: No providers have been configured and the built in memory cache has not been enabled.")
		}
		logger.Warn(fmt.Sprintf(`[caching-module]: No default caching provider defined. Using "%s" as default.`, defaultProvider))
	}
	return registry, defaultProvider, nil
}
