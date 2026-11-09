// Package utils mirrors @medusajs/inventory/src/utils.
package utils

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// ApplyEntityHooks mirrors applyEntityHooks: InventoryLevel computes
// available_quantity (stocked - reserved) when it is hydrated. The lazy
// stocked/reserved formulas of InventoryItem are not ported yet.
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

func init() { ApplyEntityHooks() }
