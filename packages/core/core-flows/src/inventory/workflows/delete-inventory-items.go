package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// DeleteInventoryItemWorkflow mirrors deleteInventoryItemWorkflow (input:
// the inventory item ids).
var DeleteInventoryItemWorkflow = composer.CreateWorkflow("delete-inventory-item-workflow",
	func(e *composer.Execution, ids []any) ([]any, error) {
		filters := dal.NewRecord()
		filters.Set("id", ids)
		res, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{GraphInput: query.GraphInput{Entity: "inventory", Fields: []string{"id", "reserved_quantity"}, Filters: filters}})
		if err != nil {
			return nil, err
		}
		items, _ := res.Data.([]any)
		if _, err := steps.ValidateInventoryDeleteStep.Run(e, items); err != nil {
			return nil, err
		}
		if _, err := steps.DeleteInventoryItemStep.Run(e, ids); err != nil {
			return nil, err
		}
		inner := dal.NewRecord()
		inner.Set("inventory_item_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.INVENTORY, inner)
		if _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}
		events := make([]any, len(ids))
		for i, id := range ids {
			ev := dal.NewRecord()
			ev.Set("id", id)
			events[i] = ev
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-item.deleted", Data: events}); err != nil {
			return nil, err
		}
		return ids, nil
	})
