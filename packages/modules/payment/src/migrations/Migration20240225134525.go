package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"

	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// Migration20240225134525 is hand-ported: upstream branches on whether the "payment_collection" table already
// exists (databases created before the module was introduced).
var Migration20240225134525 = m.Migration{
	Name: "Migration20240225134525",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		exists, err := m.TableExists(ctx, tx, "payment_collection")
		if err != nil {
			return err
		}
		if exists {
			return m.ExecAll(ctx, tx, Migration20240225134525UpWhenExists)
		}
		return m.ExecAll(ctx, tx, Migration20240225134525UpWhenMissing)
	},
	Down: Migration20240225134525Down,
}
