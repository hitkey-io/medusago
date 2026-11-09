package migrationscripts

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// reconcileInventoryReservedQuantity mirrors
// reconcile-inventory-reserved-quantity.ts: reserved quantities are aligned
// with the active reservations (one transaction instead of the
// compensable workflow steps).
func (c Context) reconcileInventoryReservedQuantity(ctx context.Context) error {
	c.Logger.Info("Starting reconciliation of inventory_level.reserved_quantity against active reservation_item rows...")
	err := pgx.BeginFunc(ctx, c.DB, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
      WITH expected AS (
        SELECT
          il.id,
          COALESCE((
            SELECT SUM(ri.quantity)
            FROM reservation_item ri
            WHERE ri.inventory_item_id = il.inventory_item_id
              AND ri.location_id = il.location_id
              AND ri.deleted_at IS NULL
          ), 0)::text AS expected_reserved_quantity,
          il.reserved_quantity::text AS current_reserved_quantity
        FROM inventory_level il
        WHERE il.deleted_at IS NULL
      )
      SELECT id, expected_reserved_quantity
      FROM expected
      WHERE current_reserved_quantity::numeric <> expected_reserved_quantity::numeric`)
		if err != nil {
			return err
		}
		type mismatch struct{ id, expected string }
		mismatches, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (mismatch, error) {
			var m mismatch
			return m, r.Scan(&m.id, &m.expected)
		})
		if err != nil {
			return err
		}
		for _, m := range mismatches {
			f, _ := strconv.ParseFloat(m.expected, 64)
			raw, _ := json.Marshal(map[string]any{"value": totals.JSNumberToPrecision(f, 20), "precision": 20})
			if _, err := tx.Exec(ctx, `update "inventory_level" set "reserved_quantity" = $1::numeric, "raw_reserved_quantity" = $2 where "id" = $3`,
				m.expected, string(raw), m.id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	c.Logger.Info("Finished reconciliation of inventory_level.reserved_quantity.")
	return nil
}
