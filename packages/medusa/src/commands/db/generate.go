package db

import (
	"context"
	"fmt"
	"github.com/hitkey-io/medusago/packages/core/framework/src/plugins"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
	migrationscripts "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk/migration-scripts"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// errGenerateReported is returned once the command logged its error (the
// CLI does not log errors wrapping ErrDatabaseMissing again).
var errGenerateReported = fmt.Errorf("migrations generation failed: %w", ErrDatabaseMissing)

// GenerateOptions are the inputs of Generate.
type GenerateOptions struct {
	// Directory is the project directory the module paths (ModuleConfig
	// Resolve) are relative to.
	Directory string
	// Modules are the keys of the modules to generate migrations for.
	Modules []string
	Logger  *slog.Logger
}

// isLocalPath tells whether a module resolve value is a path (a project
// module) rather than a package name.
func isLocalPath(resolve string) bool {
	return strings.HasPrefix(resolve, "./") || strings.HasPrefix(resolve, "../") || filepath.IsAbs(resolve) || resolve == "." || resolve == ".."
}

// Generate mirrors `medusa db:generate <modules...>` (commands/db/generate.ts
// with MedusaAppMigrateGenerate): for each module, the migration of the
// changes of its models since the snapshot of its migrations directory is
// written as Go code into <module directory>/migrations, with index.go
// listing the migrations of the directory. The module directory is the
// path of the module configuration (resolve) relative to the project
// directory: only project modules (resolved from a local path) can
// generate migrations.
func Generate(ctx context.Context, cfg config.ConfigModule, opts GenerateOptions) error {
	// the modules and links of the plugins (mergePluginModules)
	if err := plugins.Apply(&cfg); err != nil {
		return err
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	pool, err := connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := EnsureDbExists(ctx, pool, logger); err != nil {
		return err
	}

	logger.Info("Generating migrations...")

	if err := migrations.EnsureTable(ctx, pool); err != nil {
		terminalSeparator()
		return err
	}

	enabled := map[string]config.ModuleConfig{}
	var all []string
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		enabled[mc.Key] = mc
		all = append(all, mc.Key)
	}
	var missing, names []string
	seen := map[string]bool{}
	for _, name := range opts.Modules {
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
		if _, ok := enabled[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		terminalSeparator()
		logger.Error(fmt.Sprintf("Cannot generate migrations for unknown module(s) %s", strings.Join(missing, ",")))
		list := make([]string, len(all))
		for i, m := range all {
			list[i] = "          - " + m
		}
		logger.Error("Available modules:\n" + strings.Join(list, "\n"))
		return errGenerateReported
	}

	var failures []string
	for _, name := range names {
		mc := enabled[name]
		lockKey := "db-module-migration:" + name
		err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, lockKey); err != nil {
				return err
			}
			exp, ok := modules.Lookup(mc)
			if !ok {
				return nil
			}
			if !isLocalPath(mc.Resolve) {
				terminalSeparator()
				logger.Info("")
				logger.Info(fmt.Sprintf("MODULE: %s", name))
				logger.Error(fmt.Sprintf("Failed with error Cannot generate migrations for the module %s: only the modules of the project (resolved from a local path) can generate migrations", name))
				return nil
			}
			dir := mc.Resolve
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(opts.Directory, dir)
			}
			migrationscripts.BuildGenerateMigrationScript(name, exp.Models, filepath.Join(dir, "migrations"))(logger)
			return nil
		})
		if err != nil {
			failures = append(failures, err.Error())
		}
	}
	terminalSeparator()
	if len(failures) > 0 {
		logger.Error(fmt.Sprintf("Some migrations failed to generate: %s", strings.Join(failures, ", ")))
		return errGenerateReported
	}
	logger.Info("Migrations generated")
	return nil
}
