// Package workflows mirrors @medusajs/core-flows/src/payment-collection/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateRefundReasonsWorkflow mirrors createRefundReasonsWorkflow.
var CreateRefundReasonsWorkflow = composer.CreateWorkflow("create-refund-reasons-workflow",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateRefundReasonStep.Run(e, data)
	})

// UpdateRefundReasonsWorkflow mirrors updateRefundReasonsWorkflow.
var UpdateRefundReasonsWorkflow = composer.CreateWorkflow("update-refund-reasons",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		return steps.UpdateRefundReasonsStep.Run(e, data)
	})

// DeleteRefundReasonsWorkflow mirrors deleteRefundReasonsWorkflow.
var DeleteRefundReasonsWorkflow = composer.CreateWorkflow("delete-refund-reasons-workflow",
	func(e *composer.Execution, ids []any) (any, error) {
		return steps.DeleteRefundReasonsStep.Run(e, ids)
	})
