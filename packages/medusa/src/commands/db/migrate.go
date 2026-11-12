// Package db mirrors @medusajs/medusa/src/commands/db.
package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/hitkey-io/medusago/packages/core/framework/src/plugins"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/framework/src/container"
	frameworkmigrations "github.com/hitkey-io/medusago/packages/core/framework/src/migrations"
	coremodulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
	migrationscripts "github.com/hitkey-io/medusago/packages/medusa/src/migration-scripts"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

// MigrateOptions mirrors the flags of `medusa db:migrate`.
type MigrateOptions struct {
	SkipLinks         bool
	SkipScripts       bool
	SkipSearch        bool
	ExecuteAllLinks   bool
	ExecuteSafeLinks  bool
	ExecuteAllSearch  bool
	ExecuteSafeSearch bool
	// Concurrency and AllOrNothing are accepted for compatibility: modules
	// are migrated one at a time, each module in one transaction.
	Concurrency  int
	AllOrNothing bool
}

// ErrDatabaseMissing is returned by EnsureDbExists after it reported the
// problem.
var ErrDatabaseMissing = errors.New("database is not reachable")

// terminalSeparator mirrors logger.log(new Array(process.stdout.columns).join("-")):
// a line of dashes on a terminal, an empty line otherwise.
func terminalSeparator() {
	info, err := os.Stdout.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		fmt.Println()
		return
	}
	columns := 80
	if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 1 {
		columns = c
	}
	fmt.Println(strings.Repeat("-", columns-1))
}

// EnsureDbExists mirrors ensureDbExists of commands/utils.
func EnsureDbExists(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	if _, err := pool.Exec(ctx, "SELECT 1 + 1;"); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "3D000" {
			logger.Error(fmt.Sprintf("Cannot sync links. %s", pgErr.Message))
			logger.Info(`Run command "db:create" to create the database`)
		} else {
			logger.Error(err.Error())
		}
		return ErrDatabaseMissing
	}
	return nil
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
		if exp, ok := modules.Lookup(mc); ok {
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

// connect opens the pool of projectConfig.databaseUrl.
func connect(ctx context.Context, cfg config.ConfigModule) (*pgxpool.Pool, error) {
	if cfg.ProjectConfig.DatabaseURL == "" {
		return nil, fmt.Errorf("databaseUrl is not configured (set DATABASE_URL)")
	}
	return pgxpool.New(ctx, cfg.ProjectConfig.DatabaseURL)
}

// Migrate mirrors migrate(): the migrations of all enabled modules and their
// providers, the link tables, the search indexes, the module loaders (which
// upstream run when the application is loaded) and the migration scripts.
func Migrate(ctx context.Context, cfg config.ConfigModule, opts MigrateOptions) error {
	// the modules and links of the plugins (mergePluginModules)
	if err := plugins.Apply(&cfg); err != nil {
		return err
	}
	logger := slog.Default()
	pool, err := connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := EnsureDbExists(ctx, pool, logger); err != nil {
		return err
	}

	logger.Info("Running migrations...")

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
				logger.Info(fmt.Sprintf("MODULE: %s", mm.key), "migrations", len(applied))
			}
		}
	}

	terminalSeparator()
	logger.Info("Migrations completed")

	if !opts.SkipLinks {
		terminalSeparator()
		if err := SyncLinks(ctx, pool, SyncLinksOptions{ExecuteAll: opts.ExecuteAllLinks, ExecuteSafe: opts.ExecuteSafeLinks, Config: &cfg}); err != nil {
			return err
		}
	}

	if !opts.SkipSearch && IsSearchModuleEnabled(cfg) {
		terminalSeparator()
		if err := MigrateSearch(ctx, cfg, MigrateSearchOptions{ExecuteAll: opts.ExecuteAllSearch, ExecuteSafe: opts.ExecuteSafeSearch}); err != nil {
			return err
		}
	}

	// Like upstream, loading the application runs the module loaders.
	if err := RunModuleLoaders(ctx, pool, cfg); err != nil {
		return err
	}

	if !opts.SkipScripts {
		terminalSeparator()
		if err := runMigrationScripts(ctx, pool, cfg, logger); err != nil {
			return err
		}
	}
	return nil
}

// IsSearchModuleEnabled mirrors isSearchModuleEnabled of loaders/search.
func IsSearchModuleEnabled(cfg config.ConfigModule) bool {
	for _, mc := range cfg.Modules {
		if mc.Key == utilsmodulessdk.SEARCH {
			return !mc.Disable
		}
	}
	return false
}

// RunMigrationScripts mirrors `db:migrate:scripts` (runMigrationScripts):
// the application is loaded (module loaders) and the pending migration
// scripts are executed.
func RunMigrationScripts(ctx context.Context, cfg config.ConfigModule) error {
	// the modules and links of the plugins (mergePluginModules)
	if err := plugins.Apply(&cfg); err != nil {
		return err
	}
	logger := slog.Default()
	pool, err := connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := EnsureDbExists(ctx, pool, logger); err != nil {
		return err
	}
	if err := RunModuleLoaders(ctx, pool, cfg); err != nil {
		return err
	}
	return runMigrationScripts(ctx, pool, cfg, logger)
}

func runMigrationScripts(ctx context.Context, pool *pgxpool.Pool, cfg config.ConfigModule, logger *slog.Logger) error {
	scripts := migrationscripts.Scripts(migrationscripts.Context{DB: pool, Config: cfg, Logger: logger})
	pending, err := frameworkmigrations.GetPendingScripts(ctx, pool, scripts)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		logger.Info("No pending migration scripts to execute")
		return nil
	}
	terminalSeparator()
	logger.Info("Pending migration scripts to execute")
	logger.Info(strings.Join(pending, "\n"))

	terminalSeparator()
	logger.Info("Running migration scripts...")
	lock, shutdown, err := lockingModule(pool, cfg, logger)
	if err != nil {
		return err
	}
	defer shutdown()
	if err := frameworkmigrations.RunScriptsWithLock(ctx, pool, scripts, lock); err != nil {
		return err
	}
	terminalSeparator()
	logger.Info("Migration scripts completed")
	return nil
}

// RunModuleLoaders runs the loaders of the enabled modules in configuration
// order, with their configured providers.
func RunModuleLoaders(ctx context.Context, pool *pgxpool.Pool, cfg config.ConfigModule) error {
	// the modules and links of the plugins (mergePluginModules)
	if err := plugins.Apply(&cfg); err != nil {
		return err
	}
	logger := slog.Default()
	// The application is loaded: the joiner configs of the modules and
	// links are known (MedusaModule.getAllJoinerConfigs(), the index module
	// builds its schema over them).
	qc, err := loaders.BuildQueryConfig(cfg)
	if err != nil {
		return err
	}
	coremodulessdk.SetAllJoinerConfigs(qc.JoinerConfigs)
	for _, mc := range cfg.Modules {
		if mc.Disable {
			continue
		}
		exp, ok := modules.Lookup(mc)
		if !ok {
			continue
		}
		var providers []modulessdk.ProviderInstance
		for _, p := range mc.Providers {
			pexp, ok := modules.Registry[p.Resolve]
			if !ok {
				providers = append(providers, modulessdk.ProviderInstance{ID: p.ID, Options: p.Options})
				continue
			}
			for _, ps := range pexp.Services() {
				providers = append(providers, modulessdk.ProviderInstance{Identifier: ps.Identifier, ID: p.ID, Options: p.Options})
			}
		}
		for _, load := range exp.Loaders {
			if err := load(ctx, modulessdk.LoaderOptions{DB: pool, Providers: providers, Options: mc.Options, Logger: logger}); err != nil {
				return fmt.Errorf("%s loader: %w", mc.Key, err)
			}
		}
	}
	return nil
}

// lockingModule builds the configured locking module (with its providers)
// the migration scripts run under, like the container of the loaded
// application upstream. It returns nil when the module is disabled.
func lockingModule(pool *pgxpool.Pool, cfg config.ConfigModule, logger *slog.Logger) (frameworkmigrations.ScriptsLock, func(), error) {
	noop := func() {}
	for _, mc := range cfg.Modules {
		if mc.Key != utilsmodulessdk.LOCKING || mc.Disable {
			continue
		}
		exp, ok := modules.Lookup(mc)
		if !ok || exp.Service == nil {
			return nil, noop, nil
		}
		c := container.New()
		c.Register(common.ContainerRegistrationKeys.PG_CONNECTION, pool)
		c.Register(common.ContainerRegistrationKeys.LOGGER, logger)
		var providers []modulessdk.ProviderInstance
		for _, p := range mc.Providers {
			pi := modulessdk.ProviderInstance{ID: p.ID, Options: p.Options, IsDefault: p.IsDefault}
			if pexp, ok := modules.Registry[p.Resolve]; ok {
				pi.Identifier = pexp.ProviderIdentifier
				if pexp.NewProvider != nil {
					inst, err := pexp.NewProvider(modulessdk.ProviderOptions{ID: p.ID, Options: p.Options, Logger: logger, Container: c})
					if err != nil {
						return nil, noop, fmt.Errorf("%s provider %s: %w", mc.Key, p.ID, err)
					}
					pi.Instance = inst
				}
			}
			providers = append(providers, pi)
		}
		svc, err := exp.Service(modulessdk.ServiceOptions{DB: pool, Providers: providers, Options: mc.Options, Logger: logger, Container: c})
		if err != nil {
			return nil, noop, err
		}
		lock, _ := svc.(frameworkmigrations.ScriptsLock)
		shutdown := noop
		if s, ok := svc.(interface {
			OnApplicationShutdown(ctx context.Context) error
		}); ok {
			shutdown = func() { _ = s.OnApplicationShutdown(context.Background()) }
		}
		return lock, shutdown, nil
	}
	return nil, noop, nil
}
