package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// BulkCreateDeleteLevelsInput mirrors BulkCreateDeleteLevelsWorkflowInput.
type BulkCreateDeleteLevelsInput struct {
	Creates []*dal.Record `json:"creates"`
	// Deletes are { inventory_item_id, location_id } pairs.
	Deletes []any `json:"deletes"`
}

// BulkCreateDeleteLevelsWorkflow mirrors bulkCreateDeleteLevelsWorkflow:
// the levels matching any of the deletes are deleted, the creates are
// created and inventory-level.created is emitted for them. No route runs
// it.
var BulkCreateDeleteLevelsWorkflow = composer.CreateWorkflow("bulk-create-delete-levels-workflow",
	func(e *composer.Execution, input BulkCreateDeleteLevelsInput) ([]*dal.Record, error) {
		if len(input.Deletes) > 0 {
			f := dal.NewRecord()
			f.Set("$or", input.Deletes)
			if _, err := DeleteInventoryLevelsWorkflow.RunAsStep(e, DeleteInventoryLevelsInput{Filters: f}); err != nil {
				return nil, err
			}
		}
		created := []*dal.Record{}
		if len(input.Creates) > 0 {
			out, err := steps.CreateInventoryLevelsStep.Run(e, input.Creates)
			if err != nil {
				return nil, err
			}
			if out != nil {
				created = out
			}
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.created", Data: idEvents(created)}); err != nil {
			return nil, err
		}
		return created, nil
	})
