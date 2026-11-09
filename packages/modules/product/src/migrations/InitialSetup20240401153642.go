package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"

	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// InitialSetup20240401153642 is hand-ported: upstream branches on whether the "product" table already
// exists (databases created before the module was introduced).
var InitialSetup20240401153642 = m.Migration{
	Name: "InitialSetup20240401153642",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		exists, err := m.TableExists(ctx, tx, "product")
		if err != nil {
			return err
		}
		if exists {
			return m.ExecAll(ctx, tx, InitialSetup20240401153642UpWhenExists)
		}
		return m.ExecAll(ctx, tx, InitialSetup20240401153642UpWhenMissing)
	},
	Down: InitialSetup20240401153642Down,
}
