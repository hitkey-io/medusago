package db

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// MigrateSearchOptions mirrors the flags of `medusa db:migrate:search`.
type MigrateSearchOptions struct {
	ExecuteAll  bool
	ExecuteSafe bool
}

// searchMigrator is the part of the Search Module the command uses.
type searchMigrator interface {
	CreateIndexMigrationPlan(ctx context.Context) ([]searchtypes.MigrationAction, error)
	ExecuteIndexMigrationPlan(ctx context.Context, actions []searchtypes.MigrationAction) error
}

type mapResolver map[string]any

func (m mapResolver) Resolve(key string) any { return m[key] }

func describeAction(action searchtypes.MigrationAction) string {
	switch action.Action {
	case "drop":
		detail := "no physical index"
		if len(action.PhysicalNames) > 0 {
			detail = strings.Join(action.PhysicalNames, ", ")
		}
		return fmt.Sprintf("%s (%s)", action.Index, detail)
	case "migrate":
		destination := action.ActivePhysicalName + " -> " + action.PhysicalName
		if action.PreviousProvider != "" {
			destination = action.PreviousProvider + " -> " + action.Provider + ", " + destination
		}
		return fmt.Sprintf("%s (%s)", action.Index, destination)
	}
	return fmt.Sprintf("%s (%s)", action.Index, action.PhysicalName)
}

func logActions(logger *slog.Logger, title string, actions []searchtypes.MigrationAction) {
	lines := []string{title}
	for _, a := range actions {
		lines = append(lines, "  - "+describeAction(a))
	}
	logger.Info(strings.Join(lines, "\n"))
}

// MigrateSearch mirrors `db:migrate:search` (migrateSearchIndexes): brings
// the physical search indexes in line with the registered definitions.
// Indexes no definition declares any more are dropped only with
// ExecuteAll (there is no interactive prompt).
func MigrateSearch(ctx context.Context, cfg config.ConfigModule, opts MigrateSearchOptions) error {
	logger := slog.Default()
	var mc *config.ModuleConfig
	for i := range cfg.Modules {
		if cfg.Modules[i].Key == utilsmodulessdk.SEARCH && !cfg.Modules[i].Disable {
			mc = &cfg.Modules[i]
		}
	}
	if mc == nil {
		logger.Info("Search Module is not configured. Skipping search indexes")
		return nil
	}
	exp, ok := modules.Registry[mc.Resolve]
	if !ok || exp.Service == nil {
		logger.Info("Search Module is not configured. Skipping search indexes")
		return nil
	}
	pool, err := connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := EnsureDbExists(ctx, pool, logger); err != nil {
		return err
	}
	container := mapResolver{
		common.ContainerRegistrationKeys.PG_CONNECTION: pool,
		common.ContainerRegistrationKeys.LOGGER:        logger,
	}
	var providers []modulessdk.ProviderInstance
	for _, p := range mc.Providers {
		pi := modulessdk.ProviderInstance{ID: p.ID, Options: p.Options}
		if pexp, ok := modules.Registry[p.Resolve]; ok {
			pi.Identifier = pexp.ProviderIdentifier
			if pexp.NewProvider != nil {
				inst, err := pexp.NewProvider(modulessdk.ProviderOptions{ID: p.ID, Options: p.Options, Logger: logger, Container: container})
				if err != nil {
					return err
				}
				pi.Instance = inst
			}
		}
		providers = append(providers, pi)
	}
	svc, err := exp.Service(modulessdk.ServiceOptions{DB: pool, Providers: providers, Logger: logger, Container: container})
	if err != nil {
		return err
	}
	searchModule, ok := svc.(searchMigrator)
	if !ok {
		return nil
	}

	logger.Info("Migrating search indexes...")
	plan, err := searchModule.CreateIndexMigrationPlan(ctx)
	if err != nil {
		return err
	}
	var noops, toCreate, toMigrate, drops []searchtypes.MigrationAction
	for _, a := range plan {
		switch a.Action {
		case "noop":
			noops = append(noops, a)
		case "create":
			toCreate = append(toCreate, a)
		case "migrate":
			toMigrate = append(toMigrate, a)
		case "drop":
			drops = append(drops, a)
		}
	}
	var toDrop []searchtypes.MigrationAction
	if len(drops) > 0 {
		switch {
		case opts.ExecuteSafe:
			logActions(logger, "Left the following search indexes in place, which no definition declares any more (--execute-safe-search)", drops)
		case opts.ExecuteAll:
			toDrop = drops
		default:
			logActions(logger, "Left the following search indexes in place, which no definition declares any more (no prompt to answer; re-run with --execute-all-search to drop them)", drops)
		}
	}
	if len(toCreate) == 0 && len(toMigrate) == 0 && len(toDrop) == 0 {
		logger.Info("Search indexes already up-to-date")
		return nil
	}
	actions := append(append(append(append([]searchtypes.MigrationAction{}, noops...), toCreate...), toMigrate...), toDrop...)
	if err := searchModule.ExecuteIndexMigrationPlan(ctx, actions); err != nil {
		return err
	}
	if len(toCreate) > 0 {
		logActions(logger, "Created following search indexes", toCreate)
	}
	if len(toMigrate) > 0 {
		logActions(logger, "Rebuilt following search indexes", toMigrate)
	}
	if len(toDrop) > 0 {
		logActions(logger, "Dropped following search indexes", toDrop)
	}
	logger.Info("Search indexes migrated. They are filled when the application starts")
	return nil
}
