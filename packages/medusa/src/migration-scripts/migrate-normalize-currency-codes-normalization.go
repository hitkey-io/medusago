package migrationscripts

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// migrateNormalizeCurrencyCodes mirrors
// migrate-normalize-currency-codes-normalization.ts. The workflow steps run
// in one transaction: a failure rolls everything back, like their
// compensations.
func (c Context) migrateNormalizeCurrencyCodes(ctx context.Context) error {
	c.Logger.Info("Starting migration to normalize currency codes...")
	tables := []string{"cart", "payment_collection", "payment_session", "payment", "order", "order_transaction", "price", "region", "store_currency"}
	err := pgx.BeginFunc(ctx, c.DB, func(tx pgx.Tx) error {
		for _, table := range tables {
			rows, err := tx.Query(ctx, `select "id" from "`+table+`" where "currency_code" is not null and currency_code <> LOWER(currency_code)`)
			if err != nil {
				return err
			}
			ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				continue
			}
			if _, err := tx.Exec(ctx, `update "`+table+`" set "currency_code" = LOWER(currency_code) where "id" = any($1)`, ids); err != nil {
				return err
			}
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'index_data')`).Scan(&exists); err != nil {
			return err
		}
		if exists {
			_, err := tx.Exec(ctx, `
        WITH prev AS (
          SELECT id, data FROM index_data
          WHERE name = 'Price'
            AND data->>'currency_code' IS NOT NULL
            AND data->>'currency_code' <> LOWER(data->>'currency_code')
        ),
        updated AS (
          UPDATE index_data
          SET data = jsonb_set(data, '{currency_code}', to_jsonb(LOWER(data->>'currency_code')))
          WHERE id IN (SELECT id FROM prev)
          RETURNING id
        )
        SELECT id, data FROM prev`)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	c.Logger.Info("Finished migration to normalize currency codes.")
	return nil
}
