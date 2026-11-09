package migrations

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	m "github.com/hitkey-io/medusago/packages/core/utils/src/migrations"
)

// Migration20250825132614 is hand-ported: it replaces the "type-code"
// shipping option types with a single "default" type.
var Migration20250825132614 = m.Migration{
	Name: "Migration20250825132614",
	UpFunc: func(ctx context.Context, tx pgx.Tx) error {
		// 1. Find all type-code shipping option types
		rows, err := tx.Query(ctx, `
      SELECT id FROM "shipping_option_type"
      WHERE code = 'type-code' AND deleted_at IS NULL
    `)
		if err != nil {
			return err
		}
		ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil || len(ids) == 0 {
			return err
		}

		// 2. Create default shipping option type
		defaultTypeID := common.GenerateEntityID("", "sotype")
		if _, err := tx.Exec(ctx, `
        INSERT INTO "shipping_option_type" (id, label, description, code)
        VALUES ($1, 'Default', 'Default shipping option type', 'default');
      `, defaultTypeID); err != nil {
			return err
		}

		// 3. Reassign shipping options to the default type
		if _, err := tx.Exec(ctx, `
        UPDATE "shipping_option"
        SET shipping_option_type_id = $1
        WHERE shipping_option_type_id = ANY($2) AND deleted_at IS NULL;
      `, defaultTypeID, ids); err != nil {
			return err
		}

		// 4. Soft delete the old type-code types
		_, err = tx.Exec(ctx, `
        UPDATE "shipping_option_type"
        SET deleted_at = now()
        WHERE id = ANY($1);
      `, ids)
		return err
	},
	// Not reversible upstream either; down is a no-op.
	Down: []string{},
}
