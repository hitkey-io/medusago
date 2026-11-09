package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"

	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// Migration20240311145700_InitialSetupMigration is hand-ported: upstream branches on whether the "shipping_option" table already
// exists (databases created before the module was introduced).
var Migration20240311145700_InitialSetupMigration = m.Migration{
	Name: "Migration20240311145700_InitialSetupMigration",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		exists, err := m.TableExists(ctx, tx, "shipping_option")
		if err != nil {
			return err
		}
		if exists {
			return m.ExecAll(ctx, tx, Migration20240311145700_InitialSetupMigrationUpWhenExists)
		}
		return m.ExecAll(ctx, tx, Migration20240311145700_InitialSetupMigrationUpWhenMissing)
	},
	Down: Migration20240311145700_InitialSetupMigrationDown,
}
