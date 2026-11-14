package hooks

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateOrderInput is the input of createOrderWorkflow.
type CreateOrderInput struct {
	OrderID string
}

// region: create-hook
// CreateOrderWorkflow exposes the hook "orderCreated": the handler
// registered for it runs at that point, as a step of the workflow. Without a
// handler, the hook does nothing.
var CreateOrderWorkflow = composer.CreateWorkflow("create-order",
	func(e *composer.Execution, input CreateOrderInput) (string, error) {
		// TODO create the order

		hookInput := dal.NewRecord()
		hookInput.Set("order_id", input.OrderID)
		if err := composer.CreateHook(e, "orderCreated", hookInput); err != nil {
			return "", err
		}

		return input.OrderID, nil
	})

// endregion: create-hook

// region: result
// PricingWorkflow uses the value a hook handler returns.
var PricingWorkflow = composer.CreateWorkflow("pricing",
	func(e *composer.Execution, input CreateOrderInput) (*dal.Record, error) {
		hookInput := dal.NewRecord()
		hookInput.Set("order_id", input.OrderID)

		result, err := composer.CreateHookWithResult(e, "setPricingContext", hookInput)
		if err != nil {
			return nil, err
		}

		// result is nil if no handler is registered
		pricingContext, _ := result.(*dal.Record)
		return pricingContext, nil
	})

// endregion: result
