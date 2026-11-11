// Package modulessdk mirrors the module-related types of
// @medusajs/types/src/modules-sdk.
package modulessdk

import (
	"context"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dml"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// ModuleExports is what a module or module provider package exports.
type ModuleExports struct {
	// Migrations are the database migrations of the package, in order.
	Migrations []migrations.Migration
	// Models are the data models of the module.
	Models []*dml.Entity
	// JoinerConfig is the joiner configuration of the module service.
	JoinerConfig *joiner.ModuleJoinerConfig
	// Loaders run when the module is loaded (seed data, provider sync...).
	Loaders []Loader
	// ProviderIdentifier is the static identifier of a module provider.
	ProviderIdentifier string
	// Service builds the module service (nil: the generic MedusaService
	// over Models).
	Service func(opts ServiceOptions) (any, error)
	// NewProvider builds a module provider instance from its options.
	NewProvider func(opts ProviderOptions) (any, error)
	// ProviderServices are the services of a module provider package that
	// ships several (ModuleProvider(..., { services: [...] })): each is
	// registered with its own identifier. When empty, ProviderIdentifier
	// and NewProvider describe the single service.
	ProviderServices []ProviderService
	// LoadersOnStart runs the Loaders when the application starts too.
	// Upstream runs the loaders of every module whenever it is loaded; in Go
	// only the modules setting this do so far (db:migrate runs all).
	LoadersOnStart bool
	// OnApplicationStart mirrors the service's __hooks.onApplicationStart:
	// it runs with the module service once the application is loaded.
	OnApplicationStart func(ctx context.Context, service any) error
}

// Resolver resolves container registrations.
type Resolver interface {
	Resolve(key string) any
}

// ServiceOptions is what a module service constructor receives.
type ServiceOptions struct {
	DB        dal.Querier
	Providers []ProviderInstance
	Options   map[string]any
	Logger    *slog.Logger
	Container Resolver
}

// ProviderOptions is what a module provider constructor receives.
type ProviderOptions struct {
	ID        string
	Options   map[string]any
	Logger    *slog.Logger
	Container Resolver
}

// ProviderService is one service of a module provider package.
type ProviderService struct {
	Identifier string
	// ValidateOptions mirrors the static validateOptions run when the
	// provider is registered.
	ValidateOptions func(options map[string]any) error
	NewProvider     func(opts ProviderOptions) (any, error)
}

// Services returns the provider services of a module provider package.
func (e *ModuleExports) Services() []ProviderService {
	if len(e.ProviderServices) > 0 {
		return e.ProviderServices
	}
	return []ProviderService{{Identifier: e.ProviderIdentifier, NewProvider: e.NewProvider}}
}

// ProviderInstance is a module provider registered from the configuration.
type ProviderInstance struct {
	Identifier string
	ID         string
	Options    map[string]any
	// Instance is the provider built by its NewProvider (nil when the
	// provider package has none).
	Instance any
}

// LoaderOptions mirrors LoaderOptions: what a module loader receives.
type LoaderOptions struct {
	DB        dal.Querier
	Providers []ProviderInstance
	Logger    *slog.Logger
}

// Loader mirrors a module loader.
type Loader func(ctx context.Context, opts LoaderOptions) error
