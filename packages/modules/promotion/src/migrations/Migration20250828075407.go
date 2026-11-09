package migrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"

	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// Migration20250828075407 is hand-ported: it warns about buy-get promotions
// without apply_to_quantity, then copies apply_to_quantity into max_quantity.
var Migration20250828075407 = m.Migration{
	Name: "Migration20250828075407",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		var count int64
		if err := tx.QueryRow(ctx, `
      SELECT COUNT(*) as count
      FROM promotion_application_method pam
      JOIN promotion p ON pam.promotion_id = p.id
      WHERE p.type = 'buyget'
      AND pam.apply_to_quantity IS NULL
    `).Scan(&count); err != nil {
			return err
		}

		if count > 0 {
			slog.Warn(fmt.Sprintf(
				"Warning: Found %d buy-get promotions with null apply_to_quantity. These should be fixed as apply_to_quantity is required for proper buy-get promotion functionality.",
				count,
			))
		}

		// The statements are identical for both branches of the read above.
		return m.ExecAll(ctx, tx, Migration20250828075407UpWhenExists)
	},
	Down: Migration20250828075407Down,
}
