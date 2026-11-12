// Package utils mirrors @medusajs/inventory/src/utils.
package utils

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// ApplyEntityHooks mirrors applyEntityHooks: InventoryLevel computes
// available_quantity (stocked - reserved) when it is hydrated, and the lazy
// stocked_quantity / reserved_quantity formulas of InventoryItem are
// registered.
func ApplyEntityHooks() {
	dal.RegisterOnInit("InventoryLevel", func(data dal.Row) {
		stocked, okS := data["raw_stocked_quantity"]
		reserved, okR := data["raw_reserved_quantity"]
		if !okS || !okR || stocked == nil || reserved == nil {
			return
		}
		v := totals.Sub(stocked, reserved)
		raw := dal.NewRecord()
		raw.Set("value", v.String())
		raw.Set("precision", float64(20))
		data["raw_available_quantity"] = raw
	})
}

func init() {
	ApplyEntityHooks()
	applyFormulas()
}

// applyFormulas mirrors applyFormulas: the lazy reserved/stocked quantity
// sums of an inventory item over its live levels.
func applyFormulas() {
	dal.RegisterFormula("InventoryItem", dal.Formula{Property: "reserved_quantity", Before: "created_at", SQL: func(item string) string {
		return "SELECT SUM(reserved_quantity) FROM inventory_level il WHERE il.inventory_item_id = " + item + ".id AND il.deleted_at IS NULL"
	}})
	dal.RegisterFormula("InventoryItem", dal.Formula{Property: "stocked_quantity", Before: "created_at", SQL: func(item string) string {
		return "SELECT SUM(stocked_quantity) FROM inventory_level il WHERE il.inventory_item_id = " + item + ".id AND il.deleted_at IS NULL"
	}})
}
