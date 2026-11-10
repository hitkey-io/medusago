package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/settings/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreatePropertyLabelsWorkflow mirrors createPropertyLabelsWorkflow.
var CreatePropertyLabelsWorkflow = composer.CreateWorkflow("create-property-labels",
	func(e *composer.Execution, in steps.CreatePropertyLabelsStepInput) ([]*dal.Record, error) {
		return steps.CreatePropertyLabelsStep.Run(e, in)
	})

// UpdatePropertyLabelsWorkflow mirrors updatePropertyLabelsWorkflow.
var UpdatePropertyLabelsWorkflow = composer.CreateWorkflow("update-property-labels",
	func(e *composer.Execution, in steps.UpdatePropertyLabelsStepInput) ([]*dal.Record, error) {
		return steps.UpdatePropertyLabelsStep.Run(e, in)
	})

// DeletePropertyLabelsWorkflow mirrors deletePropertyLabelsWorkflow.
var DeletePropertyLabelsWorkflow = composer.CreateWorkflow("delete-property-labels",
	func(e *composer.Execution, in steps.DeletePropertyLabelsStepInput) (any, error) {
		_, err := steps.DeletePropertyLabelsStep.Run(e, in)
		return nil, err
	})

// BatchPropertyLabelsWorkflowInput mirrors BatchPropertyLabelsWorkflowInput.
type BatchPropertyLabelsWorkflowInput struct {
	Create []*dal.Record
	Update []*dal.Record
	Delete []any
}

// BatchPropertyLabelsWorkflowOutput mirrors BatchPropertyLabelsWorkflowOutput.
type BatchPropertyLabelsWorkflowOutput struct {
	Created []*dal.Record
	Updated []*dal.Record
	Deleted []any
}

// BatchPropertyLabelsWorkflow mirrors batchPropertyLabelsWorkflow: the
// create, update and delete workflows run (as steps) when their input is
// not empty.
var BatchPropertyLabelsWorkflow = composer.CreateWorkflow("batch-property-labels",
	func(e *composer.Execution, in BatchPropertyLabelsWorkflowInput) (BatchPropertyLabelsWorkflowOutput, error) {
		out := BatchPropertyLabelsWorkflowOutput{Created: []*dal.Record{}, Updated: []*dal.Record{}, Deleted: []any{}}
		err := composer.Parallelize(
			func() error {
				if len(in.Create) == 0 {
					return nil
				}
				created, err := CreatePropertyLabelsWorkflow.RunAsStep(e, steps.CreatePropertyLabelsStepInput{PropertyLabels: in.Create})
				if created != nil {
					out.Created = created
				}
				return err
			},
			func() error {
				if len(in.Update) == 0 {
					return nil
				}
				updated, err := UpdatePropertyLabelsWorkflow.RunAsStep(e, steps.UpdatePropertyLabelsStepInput{PropertyLabels: in.Update})
				if updated != nil {
					out.Updated = updated
				}
				return err
			},
			func() error {
				if len(in.Delete) == 0 {
					return nil
				}
				_, err := DeletePropertyLabelsWorkflow.RunAsStep(e, steps.DeletePropertyLabelsStepInput{IDs: in.Delete})
				return err
			},
		)
		if err != nil {
			return BatchPropertyLabelsWorkflowOutput{}, err
		}
		if in.Delete != nil {
			out.Deleted = in.Delete
		}
		return out, nil
	})
