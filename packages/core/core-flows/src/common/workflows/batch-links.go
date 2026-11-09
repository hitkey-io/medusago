package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// BatchLinksInput mirrors BatchWorkflowInput<LinkDefinition>.
type BatchLinksInput struct {
	Create []*dal.Record
	Update []*dal.Record
	Delete []*dal.Record
}

// BatchLinksResult mirrors { created, updated, deleted }.
type BatchLinksResult struct {
	Created []*dal.Record
	Updated []*dal.Record
	Deleted []*dal.Record
}

// BatchLinksWorkflow mirrors batchLinksWorkflow.
var BatchLinksWorkflow = composer.CreateWorkflow("batch-links",
	func(e *composer.Execution, input BatchLinksInput) (BatchLinksResult, error) {
		var out BatchLinksResult
		err := composer.Parallelize(
			func() error {
				var err error
				out.Created, err = steps.CreateRemoteLinkStep.Run(e, nonNil(input.Create))
				return err
			},
			func() error {
				var err error
				out.Updated, err = steps.UpdateRemoteLinksStep.Run(e, nonNil(input.Update))
				return err
			},
			func() error {
				var err error
				out.Deleted, err = steps.DismissRemoteLinkStep.Run(e, nonNil(input.Delete))
				return err
			},
		)
		return out, err
	})

func nonNil(l []*dal.Record) []*dal.Record {
	if l == nil {
		return []*dal.Record{}
	}
	return l
}
