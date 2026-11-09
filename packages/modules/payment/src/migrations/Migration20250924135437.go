package migrations

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// Migration20250924135437 is hand-ported: it seeds the default refund reasons
// when the refund_reason table is empty.
var Migration20250924135437 = m.Migration{
	Name: "Migration20250924135437",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		var exists int
		err := tx.QueryRow(ctx, `
      SELECT 1 as exists
      FROM "refund_reason"
      LIMIT 1
    `).Scan(&exists)
		if err == nil {
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		_, err = tx.Exec(ctx, `
        INSERT INTO "refund_reason" (id, label, description)
        VALUES
          ($1, 'Shipping Issue', 'Refund due to lost, delayed, or misdelivered shipment'),
          ($2, 'Customer Care Adjustment', 'Refund given as goodwill or compensation for inconvenience'),
          ($3, 'Pricing Error', 'Refund to correct an overcharge, missing discount, or incorrect price');
      `,
			common.GenerateEntityID("", "refr"),
			common.GenerateEntityID("", "refr"),
			common.GenerateEntityID("", "refr"),
		)
		return err
	},
	Down: []string{
		`
      DELETE FROM "refund_reason"
      WHERE ("label", "description") IN (
        ('Shipping Issue', 'Refund due to lost, delayed, or misdelivered shipment'),
        ('Customer Care Adjustment', 'Refund given as goodwill or compensation for inconvenience'),
        ('Pricing Error', 'Refund to correct an overcharge, missing discount, or incorrect price')
      )
    `,
	},
}
