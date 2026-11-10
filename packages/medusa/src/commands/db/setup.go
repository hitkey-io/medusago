package db

import (
	"context"
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
)

// SetupOptions mirrors the flags of `medusa db:setup`.
type SetupOptions struct {
	CreateOptions
	MigrateOptions
	// LoadConfig reloads the configuration after the database is created
	// (upstream initializes the container again).
	LoadConfig func() (config.ConfigModule, error)
}

// Setup mirrors `medusa db:setup`: db:create, then db:migrate.
func Setup(ctx context.Context, opts SetupOptions) bool {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	if !Create(ctx, opts.CreateOptions) {
		return false
	}
	cfg, err := opts.LoadConfig()
	if err != nil {
		logger.Error(err.Error())
		return false
	}
	if err := Migrate(ctx, cfg, opts.MigrateOptions); err != nil {
		if err != ErrDatabaseMissing {
			logger.Error(err.Error())
		}
		return false
	}
	return true
}
