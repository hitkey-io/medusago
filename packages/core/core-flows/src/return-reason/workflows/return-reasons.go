// Package workflows mirrors @medusajs/core-flows/src/return-reason/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/return-reason/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateReturnReasonsWorkflow mirrors createReturnReasonsWorkflow.
var CreateReturnReasonsWorkflow = composer.CreateWorkflow("create-return-reasons",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateReturnReasonsStep.Run(e, data)
	})

// UpdateReturnReasonsWorkflow mirrors updateReturnReasonsWorkflow.
var UpdateReturnReasonsWorkflow = composer.CreateWorkflow("update-return-reasons",
	func(e *composer.Execution, in steps.UpdateReturnReasonsStepInput) ([]*dal.Record, error) {
		return steps.UpdateReturnReasonsStep.Run(e, in)
	})

// DeleteReturnReasonsWorkflow mirrors deleteReturnReasonsWorkflow.
var DeleteReturnReasonsWorkflow = composer.CreateWorkflow("delete-return-reasons",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteReturnReasonStep.Run(e, ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("return_reason_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.ORDER, keys)
		_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry})
		return nil, err
	})
