// Package workflows mirrors @medusajs/core-flows/src/inventory/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(rows []*dal.Record) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		id, _ := r.Get("id")
		e := dal.NewRecord()
		e.Set("id", id)
		out[i] = e
	}
	return out
}

// CreateInventoryLevelsWorkflow mirrors createInventoryLevelsWorkflow.
var CreateInventoryLevelsWorkflow = composer.CreateWorkflow("create-inventory-levels-workflow",
	func(e *composer.Execution, levels []*dal.Record) ([]*dal.Record, error) {
		if _, err := steps.ValidateInventoryLocationsStep.Run(e, levels); err != nil {
			return nil, err
		}
		created, err := steps.CreateInventoryLevelsStep.Run(e, levels)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		return created, nil
	})

// CreateInventoryItemsWorkflow mirrors createInventoryItemsWorkflow.
var CreateInventoryItemsWorkflow = composer.CreateWorkflow("create-inventory-items-workflow",
	func(e *composer.Execution, items []*dal.Record) ([]*dal.Record, error) {
		locationLevels := make([][]any, len(items))
		var inventoryItems []*dal.Record
		for i, it := range items {
			lv, _ := it.Get("location_levels")
			locationLevels[i], _ = lv.([]any)
			c := dal.NewRecord()
			for _, k := range it.Keys() {
				if k != "location_levels" {
					v, _ := it.Get(k)
					c.Set(k, v)
				}
			}
			inventoryItems = append(inventoryItems, c)
		}
		created, err := steps.CreateInventoryItemsStep.Run(e, inventoryItems)
		if err != nil {
			return nil, err
		}
		var levels []*dal.Record
		for i, item := range created {
			id, _ := item.Get("id")
			for _, l := range locationLevels[i] {
				lr, _ := l.(*dal.Record)
				c := dal.NewRecord()
				for _, k := range lr.Keys() {
					v, _ := lr.Get(k)
					c.Set(k, v)
				}
				c.Set("inventory_item_id", id)
				levels = append(levels, c)
			}
		}
		if _, err := CreateInventoryLevelsWorkflow.RunAsStep(e, levels); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-item.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		return created, nil
	})

// The Go bodies take one property of the upstream input object.
func init() {
	CreateInventoryLevelsWorkflow.InputFrom("inventory_levels")
	CreateInventoryItemsWorkflow.InputFrom("items")
}
