// Package db mirrors @medusajs/medusa/src/commands/db.
package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	frameworkmigrations "github.com/hitkey-io/medusago/packages/core/framework/src/migrations"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
	migrationscripts "github.com/hitkey-io/medusago/packages/medusa/src/migration-scripts"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// MigrateOptions mirrors the flags of `medusa db:migrate`.
type MigrateOptions struct {
	SkipLinks        bool
	SkipScripts      bool
	ExecuteAllLinks  bool
	ExecuteSafeLinks bool
}

// moduleMigrations is the migration set of one enabled module: the module's
// own migrations followed by those of its providers.
type moduleMigrations struct {
	key  string
	sets [][]migrations.Migration
}

func resolveMigrations(cfg config.ConfigModule) ([]moduleMigrations, error) {
	var result []moduleMigrations
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		mm := moduleMigrations{key: mc.Key}
		if exp, ok := modules.Registry[mc.Resolve]; ok {
			mm.sets = append(mm.sets, exp.Migrations)
		}
		for _, p := range mc.Providers {
			if exp, ok := modules.Registry[p.Resolve]; ok {
				mm.sets = append(mm.sets, exp.Migrations)
			}
		}
		result = append(result, mm)
	}
	return result, nil
}

// Migrate runs the migrations of all enabled modules and their providers.
func Migrate(ctx context.Context, cfg config.ConfigModule, opts MigrateOptions) error {
	if cfg.ProjectConfig.DatabaseURL == "" {
		return fmt.Errorf("databaseUrl is not configured (set DATABASE_URL)")
	}

	pool, err := pgxpool.New(ctx, cfg.ProjectConfig.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	slog.Info("Running migrations...")

	resolved, err := resolveMigrations(cfg)
	if err != nil {
		return err
	}

	for _, mm := range resolved {
		for _, set := range mm.sets {
			applied, err := migrations.Run(ctx, pool, mm.key, set)
			if err != nil {
				return err
			}
			if len(applied) > 0 {
				slog.Info(fmt.Sprintf("MODULE: %s", mm.key), "migrations", len(applied))
			}
		}
	}

	slog.Info("Migrations completed")

	if !opts.SkipLinks {
		if err := SyncLinks(ctx, pool, SyncLinksOptions{ExecuteAll: opts.ExecuteAllLinks, ExecuteSafe: opts.ExecuteSafeLinks}); err != nil {
			return err
		}
	}

	// Like upstream, loading the application runs the module loaders.
	if err := RunModuleLoaders(ctx, pool, cfg); err != nil {
		return err
	}

	if !opts.SkipScripts {
		slog.Info("Running migration scripts...")
		scripts := migrationscripts.Scripts(migrationscripts.Context{DB: pool, Config: cfg, Logger: slog.Default()})
		if err := frameworkmigrations.RunScripts(ctx, pool, scripts); err != nil {
			return err
		}
		slog.Info("Migration scripts completed")
	}
	return nil
}

// RunModuleLoaders runs the loaders of the enabled modules in configuration
// order, with their configured providers.
func RunModuleLoaders(ctx context.Context, pool *pgxpool.Pool, cfg config.ConfigModule) error {
	logger := slog.Default()
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		exp, ok := modules.Registry[mc.Resolve]
		if !ok {
			continue
		}
		var providers []modulessdk.ProviderInstance
		for _, p := range mc.Providers {
			identifier := ""
			if pexp, ok := modules.Registry[p.Resolve]; ok {
				identifier = pexp.ProviderIdentifier
			}
			providers = append(providers, modulessdk.ProviderInstance{Identifier: identifier, ID: p.ID, Options: p.Options})
		}
		for _, load := range exp.Loaders {
			if err := load(ctx, modulessdk.LoaderOptions{DB: pool, Providers: providers, Logger: logger}); err != nil {
				return fmt.Errorf("%s loader: %w", mc.Key, err)
			}
		}
	}
	return nil
}
