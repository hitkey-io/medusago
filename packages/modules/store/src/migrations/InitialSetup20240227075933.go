package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"

	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// InitialSetup20240227075933 is hand-ported: upstream branches on whether the "store" table already
// exists (databases created before the module was introduced).
var InitialSetup20240227075933 = m.Migration{
	Name: "InitialSetup20240227075933",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		exists, err := m.TableExists(ctx, tx, "store")
		if err != nil {
			return err
		}
		if exists {
			return m.ExecAll(ctx, tx, InitialSetup20240227075933UpWhenExists)
		}
		return m.ExecAll(ctx, tx, InitialSetup20240227075933UpWhenMissing)
	},
	Down: InitialSetup20240227075933Down,
}
