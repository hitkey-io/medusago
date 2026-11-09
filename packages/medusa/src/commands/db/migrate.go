// Package db mirrors @medusajs/medusa/src/commands/db.
package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
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
	return nil
}
