// Package workflows mirrors @medusajs/core-flows/src/user/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/user/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateUsersWorkflowID mirrors updateUsersWorkflowId.
const UpdateUsersWorkflowID = "update-users-workflow"

// UpdateUsersWorkflowInput mirrors UpdateUsersWorkflowInputDTO.
type UpdateUsersWorkflowInput struct {
	Updates []*dal.Record
}

// UpdateUsersWorkflow mirrors updateUsersWorkflow.
var UpdateUsersWorkflow = composer.CreateWorkflow(UpdateUsersWorkflowID,
	func(e *composer.Execution, input UpdateUsersWorkflowInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateUsersStep.Run(e, input.Updates)
		if err != nil {
			return nil, err
		}
		events := make([]any, len(updated))
		for i, u := range updated {
			id, _ := u.Get("id")
			r := dal.NewRecord()
			r.Set("id", id)
			events[i] = r
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "user.updated", Data: events}); err != nil {
			return nil, err
		}
		return updated, nil
	})
