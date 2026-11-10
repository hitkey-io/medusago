// Package migrations mirrors @medusajs/framework/src/migrations: the runner
// of the migration scripts, tracked in the script_migrations table.
package migrations

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationTableName = "script_migrations"

// Script is a migration script. Name is the upstream file name (with its
// .js extension), as recorded in script_migrations.
type Script struct {
	Name string
	Run  func(ctx context.Context) error
}

// hashStringToInt mirrors #hashStringToInt: a signed 32-bit hash used as the
// advisory lock key.
func hashStringToInt(s string) int32 {
	codes := []rune(s)
	h := int32(5381)
	for i := len(codes) - 1; i >= 0; i-- {
		h = int32(int64(h)*33) ^ int32(codes[i])
	}
	return h
}

func ensureMigrationsTable(ctx context.Context, db *pgxpool.Pool) error {
	var exists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = '`+migrationTableName+`')`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	slog.Info(fmt.Sprintf("Creating migrations table '%s'...", migrationTableName))
	_, err := db.Exec(ctx, `
      CREATE TABLE IF NOT EXISTS `+migrationTableName+` (
        id SERIAL PRIMARY KEY,
        script_name VARCHAR(255) NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        finished_at TIMESTAMP WITH TIME ZONE
      );

      CREATE UNIQUE INDEX IF NOT EXISTS idx_script_name_unique ON `+migrationTableName+` (script_name);
    `)
	if err == nil {
		slog.Info("Migrations table created successfully")
	}
	return err
}

// RunScripts mirrors MigrationScriptsMigrator.run: every script without a
// finished record is claimed (session advisory lock on a dedicated
// connection, so a crashed run is picked up again), run, and marked as
// finished.
func RunScripts(ctx context.Context, db *pgxpool.Pool, scripts []Script) error {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return err
	}
	finished, err := finishedScripts(ctx, db)
	if err != nil {
		return err
	}

	for _, script := range scripts {
		if finished[script.Name] {
			continue
		}
		if err := runScript(ctx, db, script); err != nil {
			return err
		}
	}
	return nil
}

// finishedScripts returns the names of the scripts with a finished record.
func finishedScripts(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	finished := map[string]bool{}
	rows, err := db.Query(ctx, `SELECT script_name, finished_at FROM `+migrationTableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var at *time.Time
		if err := rows.Scan(&name, &at); err != nil {
			return nil, err
		}
		if at != nil {
			finished[name] = true
		}
	}
	return finished, rows.Err()
}

// GetPendingScripts mirrors ensureMigrationsTable + getPendingMigrations:
// the names of the scripts that have not finished yet.
func GetPendingScripts(ctx context.Context, db *pgxpool.Pool, scripts []Script) ([]string, error) {
	if err := ensureMigrationsTable(ctx, db); err != nil {
		return nil, err
	}
	finished, err := finishedScripts(ctx, db)
	if err != nil {
		return nil, err
	}
	var pending []string
	for _, s := range scripts {
		if !finished[s.Name] {
			pending = append(pending, s.Name)
		}
	}
	return pending, nil
}

func runScript(ctx context.Context, db *pgxpool.Pool, script Script) error {
	conn, err := db.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	lockKey := hashStringToInt(migrationTableName + ":" + script.Name)
	var acquired bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1) AS acquired", lockKey).Scan(&acquired); err != nil {
		return err
	}
	if !acquired {
		slog.Info(fmt.Sprintf("Skipping migration script %s. It is being executed by another process", script.Name))
		return nil
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey) //nolint:errcheck

	var finishedAt *time.Time
	err = conn.QueryRow(ctx, `SELECT finished_at FROM `+migrationTableName+` WHERE script_name = $1`, script.Name).Scan(&finishedAt)
	exists := err == nil
	if exists && finishedAt != nil {
		return nil
	}
	if _, err := conn.Exec(ctx, `INSERT INTO `+migrationTableName+` (script_name) VALUES ($1)
         ON CONFLICT (script_name) DO UPDATE SET created_at = NOW()
         WHERE `+migrationTableName+`.finished_at IS NULL`, script.Name); err != nil {
		return err
	}
	if exists {
		slog.Warn(fmt.Sprintf("Migration script %s did not complete during a previous run and is being executed again", script.Name))
	}

	slog.Info(fmt.Sprintf("Running migration script %s", script.Name))
	start := time.Now()
	if err := script.Run(ctx); err != nil {
		slog.Error(fmt.Sprintf("Failed to run migration script %s: %v", script.Name, err))
		return err
	}
	if _, err := db.Exec(ctx, `UPDATE `+migrationTableName+` SET finished_at = NOW() WHERE script_name = $1`, script.Name); err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Migration script %s completed (%.2fs)", script.Name, time.Since(start).Seconds()))
	return nil
}
