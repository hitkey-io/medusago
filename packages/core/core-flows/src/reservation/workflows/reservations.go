// Package workflows mirrors @medusajs/core-flows/src/reservation/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/reservation/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(ids []any) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		ev := dal.NewRecord()
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

// CreateReservationsWorkflow mirrors createReservationsWorkflow.
var CreateReservationsWorkflow = composer.CreateWorkflow("create-reservations-workflow",
	func(e *composer.Execution, reservations []*dal.Record) ([]*dal.Record, error) {
		created, err := steps.CreateReservationsStep.Run(e, reservations)
		if err != nil {
			return nil, err
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.created", Data: idEvents(commonsteps.RecordIDs(created))})
		return created, err
	})

// UpdateReservationsWorkflow mirrors updateReservationsWorkflow.
var UpdateReservationsWorkflow = composer.CreateWorkflow("update-reservations-workflow",
	func(e *composer.Execution, updates []*dal.Record) ([]*dal.Record, error) {
		updated, err := steps.UpdateReservationsStep.Run(e, updates)
		if err != nil {
			return nil, err
		}
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.updated", Data: idEvents(commonsteps.RecordIDs(updated))})
		return updated, err
	})

// DeleteReservationsWorkflow mirrors deleteReservationsWorkflow.
var DeleteReservationsWorkflow = composer.CreateWorkflow("delete-reservations",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteReservationsStep.Run(e, ids); err != nil {
			return nil, err
		}
		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.deleted", Data: idEvents(ids)})
		return nil, err
	})
