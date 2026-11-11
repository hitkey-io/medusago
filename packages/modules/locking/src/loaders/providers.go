// Package loaders mirrors @medusajs/locking/src/loaders.
package loaders

import (
	"fmt"
	"log/slog"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/providers"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/services"
)

// LoadProviders mirrors the providers loader: the in-memory provider is
// always registered and is the default unless a configured provider is
// is_default or the only one. It returns the registry and the default id.
func LoadProviders(configured []modulessdk.ProviderInstance, logger *slog.Logger) (*services.LockingProviderService, string, error) {
	if logger == nil {
		logger = slog.Default()
	}
	registry := services.NewLockingProviderService(logger)
	registry.Register(providers.InMemoryIdentifier, providers.NewInMemoryLockingProvider())
	defaultProvider := providers.InMemoryIdentifier

	for _, p := range configured {
		if p.ID == "" {
			return nil, "", fmt.Errorf(`No "id" provided for provider %s`, p.Identifier)
		}
		inst, ok := p.Instance.(lockingtypes.ILockingProvider)
		if !ok {
			return nil, "", fmt.Errorf("locking provider %q is not a locking provider", p.ID)
		}
		registry.Register(p.ID, inst)
	}

	isSingleProvider := len(configured) == 1
	hasDefaultProvider := false
	for _, p := range configured {
		if p.IsDefault || isSingleProvider {
			if p.IsDefault {
				hasDefaultProvider = true
			}
			defaultProvider = p.ID
		}
	}
	if !hasDefaultProvider {
		logger.Info(fmt.Sprintf(`Locking module: Using "%s" as default.`, defaultProvider))
	}
	return registry, defaultProvider, nil
}
