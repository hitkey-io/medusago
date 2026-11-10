package db

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/definitions"
	"github.com/hitkey-io/medusago/packages/modules/link-modules/src/migration"
)

// SyncLinksOptions mirrors the flags of `medusa db:sync-links`.
type SyncLinksOptions struct {
	// ExecuteAll also applies delete actions (dropping tables of removed links).
	ExecuteAll bool
	// ExecuteSafe skips delete and notify actions without asking.
	ExecuteSafe bool
}

// SyncLinksCommand mirrors `medusa db:sync-links`.
func SyncLinksCommand(ctx context.Context, cfg config.ConfigModule, opts SyncLinksOptions) error {
	pool, err := connect(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := EnsureDbExists(ctx, pool, slog.Default()); err != nil {
		return err
	}
	return SyncLinks(ctx, pool, opts)
}

// SyncLinks mirrors syncLinks from sync-links.ts. There is no interactive
// prompt: unless ExecuteAll is set, delete and notify actions are reported and
// skipped, which is what upstream does with --execute-safe.
func SyncLinks(ctx context.Context, db *pgxpool.Pool, opts SyncLinksOptions) error {
	slog.Info("Syncing links...")

	planner, err := migration.NewPlanner(db, definitions.All)
	if err != nil {
		return err
	}
	plan, err := planner.CreatePlan(ctx)
	if err != nil {
		return err
	}

	var toExecute, created, deleted []migration.Action
	for _, a := range plan {
		switch a.Action {
		case "create", "update":
			toExecute = append(toExecute, a)
			if a.Action == "create" {
				created = append(created, a)
			}
		case "delete":
			if opts.ExecuteAll {
				toExecute = append(toExecute, a)
				deleted = append(deleted, a)
			} else {
				slog.Warn(fmt.Sprintf("Skipping removal of link table %s (use --execute-all to drop it)", a.TableName))
			}
		case "notify":
			slog.Warn(fmt.Sprintf("Link table %s differs from its definition and was not changed: %s", a.TableName, a.Reason))
		}
	}

	if err := planner.ExecutePlan(ctx, toExecute); err != nil {
		return err
	}

	for _, a := range created {
		slog.Info(fmt.Sprintf("Created link table %s", a.TableName))
	}
	for _, a := range deleted {
		slog.Info(fmt.Sprintf("Deleted link table %s", a.TableName))
	}
	if len(toExecute) > 0 {
		slog.Info("Links sync completed")
	} else {
		slog.Info("Database already up-to-date")
	}
	return nil
}
