// Package migrations mirrors @medusajs/utils/src/migrations: it runs module
// migrations and records them in the mikro_orm_migrations table exactly like
// the MikroORM migrator used by Medusa, so a database migrated by either
// implementation is recognized by the other.
package migrations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// TableName is the MikroORM migrations table shared by all modules.
const TableName = "mikro_orm_migrations"

// StepFunc is the body of a migration that has to read the database before
// deciding what to execute (hand-ported from upstream).
type StepFunc func(ctx context.Context, tx pgx.Tx) error

// Migration is a single module migration.
//
// Up and Down hold the statements captured from the upstream migration, in
// order. A nil Down means the migration cannot be reverted. UpFunc/DownFunc
// replace Up/Down for migrations with runtime logic.
type Migration struct {
	Name     string
	Up       []string
	Down     []string
	UpFunc   StepFunc
	DownFunc StepFunc
}

// ErrIrreversible is returned when reverting a migration without a down step.
var ErrIrreversible = errors.New("This migration cannot be reverted")

func (m Migration) up(ctx context.Context, tx pgx.Tx) error {
	if m.UpFunc != nil {
		return m.UpFunc(ctx, tx)
	}
	return ExecAll(ctx, tx, m.Up)
}

func (m Migration) down(ctx context.Context, tx pgx.Tx) error {
	if m.DownFunc != nil {
		return m.DownFunc(ctx, tx)
	}
	if m.Down == nil {
		return ErrIrreversible
	}
	return ExecAll(ctx, tx, m.Down)
}

// GeneratedULIDMarker stands for a value upstream generates with ulid() while
// the migration runs; every occurrence is replaced with a fresh ULID.
const GeneratedULIDMarker = "__MEDUSA_GENERATED_ULID__"

func expandGenerated(sql string) string {
	for strings.Contains(sql, GeneratedULIDMarker) {
		sql = strings.Replace(sql, GeneratedULIDMarker, common.GenerateEntityID("", ""), 1)
	}
	return sql
}

// ExecAll executes the statements in order. Each entry may contain several
// SQL statements; they are sent with the simple protocol.
func ExecAll(ctx context.Context, tx pgx.Tx, statements []string) error {
	for _, sql := range statements {
		if _, err := tx.Exec(ctx, expandGenerated(sql), pgx.QueryExecModeSimpleProtocol); err != nil {
			return err
		}
	}
	return nil
}

// EnsureTable creates the migrations table the way MikroORM does.
func EnsureTable(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `create table if not exists "`+TableName+`" ("id" serial primary key, "name" varchar(255), "executed_at" timestamptz default CURRENT_TIMESTAMP)`)
	return err
}

// Executed returns the names of the migrations already recorded.
func Executed(ctx context.Context, db *pgxpool.Pool) (map[string]bool, error) {
	rows, err := db.Query(ctx, `select "name" from "`+TableName+`" order by "id" asc`)
	if err != nil {
		return nil, err
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	executed := make(map[string]bool, len(names))
	for _, n := range names {
		executed[n] = true
	}
	return executed, nil
}

// Run applies the pending migrations of one module. Like MikroORM's
// allOrNothing mode, all pending migrations of the module run in a single
// transaction. It returns the names of the applied migrations.
func Run(ctx context.Context, db *pgxpool.Pool, module string, migrations []Migration) ([]string, error) {
	if err := EnsureTable(ctx, db); err != nil {
		return nil, err
	}
	executed, err := Executed(ctx, db)
	if err != nil {
		return nil, err
	}

	var pending []Migration
	for _, m := range migrations {
		if !executed[m.Name] {
			pending = append(pending, m)
		}
	}
	if len(pending) == 0 {
		return nil, nil
	}

	var applied []string
	err = pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		for _, m := range pending {
			if err := m.up(ctx, tx); err != nil {
				return fmt.Errorf("%s: migration %s failed: %w", module, m.Name, err)
			}
			if _, err := tx.Exec(ctx, `insert into "`+TableName+`" ("name") values ($1)`, m.Name); err != nil {
				return err
			}
			applied = append(applied, m.Name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return applied, nil
}

// Revert rolls back the last `steps` executed migrations of the module, in
// reverse order, in a single transaction.
func Revert(ctx context.Context, db *pgxpool.Pool, module string, migrations []Migration, steps int) ([]string, error) {
	executed, err := Executed(ctx, db)
	if err != nil {
		return nil, err
	}

	var toRevert []Migration
	for i := len(migrations) - 1; i >= 0 && len(toRevert) < steps; i-- {
		if executed[migrations[i].Name] {
			toRevert = append(toRevert, migrations[i])
		}
	}

	var reverted []string
	err = pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
		for _, m := range toRevert {
			if err := m.down(ctx, tx); err != nil {
				return fmt.Errorf("%s: reverting %s failed: %w", module, m.Name, err)
			}
			if _, err := tx.Exec(ctx, `delete from "`+TableName+`" where "name" = $1`, m.Name); err != nil {
				return err
			}
			reverted = append(reverted, m.Name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return reverted, nil
}

// TableExists reports whether a table exists in the public schema; used by
// hand-ported migrations that branch on the existing schema.
func TableExists(ctx context.Context, tx pgx.Tx, table string) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx,
		`select exists (select 1 from information_schema.tables where table_name = $1 and table_schema = 'public')`,
		table,
	).Scan(&exists)
	return exists, err
}
