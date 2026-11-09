// Package workflows mirrors @medusajs/core-flows/src/common/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateLinksWorkflow mirrors createLinksWorkflow.
var CreateLinksWorkflow = composer.CreateWorkflow("create-link",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateRemoteLinkStep.Run(e, input)
	})
