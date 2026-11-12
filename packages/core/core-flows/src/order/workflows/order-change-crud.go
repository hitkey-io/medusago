package workflows

import (
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// The order change workflows wrapping a single step (cancel-order-change.ts,
// create-order-change.ts, decline-order-change.ts, delete-order-change.ts,
// delete-order-change-actions.ts, update-order-changes.ts,
// update-order-change-actions.ts upstream). No route runs them: they are
// public API for custom code and the run endpoint.

// CancelOrderChangeWorkflow mirrors cancelOrderChangeWorkflow.
var CancelOrderChangeWorkflow = composer.CreateWorkflow("cancel-order-change",
	func(e *composer.Execution, input *dal.Record) (any, error) {
		_, err := ordersteps.CancelOrderChangeStep.Run(e, input)
		return nil, err
	})

// CreateOrderChangeWorkflow mirrors createOrderChangeWorkflow.
var CreateOrderChangeWorkflow = composer.CreateWorkflow("create-order-change",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		return ordersteps.CreateOrderChangeStep.Run(e, input)
	})

// DeclineOrderChangeWorkflow mirrors declineOrderChangeWorkflow.
var DeclineOrderChangeWorkflow = composer.CreateWorkflow("decline-order-change",
	func(e *composer.Execution, input *dal.Record) (any, error) {
		_, err := ordersteps.DeclineOrderChangeStep.Run(e, input)
		return nil, err
	})

// DeleteOrderChangeIDsInput mirrors DeleteOrderChangeWorkflowInput and
// DeleteOrderChangeActionsWorkflowInput ({ ids }).
type DeleteOrderChangeIDsInput struct {
	IDs []any `json:"ids"`
}

// DeleteOrderChangeWorkflow mirrors deleteOrderChangeWorkflow.
var DeleteOrderChangeWorkflow = composer.CreateWorkflow("delete-order-change",
	func(e *composer.Execution, input DeleteOrderChangeIDsInput) (any, error) {
		_, err := ordersteps.DeleteOrderChangesStep.Run(e, input.IDs)
		return nil, err
	})

// DeleteOrderChangeActionsWorkflow mirrors deleteOrderChangeActionsWorkflow.
var DeleteOrderChangeActionsWorkflow = composer.CreateWorkflow("delete-order-change-actions",
	func(e *composer.Execution, input DeleteOrderChangeIDsInput) (any, error) {
		_, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, input.IDs)
		return nil, err
	})

// UpdateOrderChangesWorkflow mirrors updateOrderChangesWorkflow.
var UpdateOrderChangesWorkflow = composer.CreateWorkflow("update-order-change",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return ordersteps.UpdateOrderChangesStep.Run(e, input)
	})

// UpdateOrderChangeActionsWorkflow mirrors updateOrderChangeActionsWorkflow.
var UpdateOrderChangeActionsWorkflow = composer.CreateWorkflow("update-order-change-actions",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return ordersteps.UpdateOrderChangeActionsStep.Run(e, input)
	})
