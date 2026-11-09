package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateLinksWorkflow mirrors updateLinksWorkflow.
var UpdateLinksWorkflow = composer.CreateWorkflow("update-link",
	func(e *composer.Execution, links []*dal.Record) ([]*dal.Record, error) {
		return steps.UpdateRemoteLinksStep.Run(e, links)
	})

// DismissLinksWorkflow mirrors dismissLinksWorkflow.
var DismissLinksWorkflow = composer.CreateWorkflow("dismiss-link",
	func(e *composer.Execution, links []*dal.Record) ([]*dal.Record, error) {
		return steps.DismissRemoteLinkStep.Run(e, links)
	})
