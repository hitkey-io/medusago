// Package locks holds the examples of the Locking Operations in Workflows
// chapter.
package locks

import (
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// WorkflowInput is the input of the workflow.
type WorkflowInput struct {
	CustomerID string `json:"customer_id"`
	OrderID    string `json:"order_id"`
}

// chargeCustomerStep stands for a step that must not run concurrently for the
// same order.
var chargeCustomerStep = composer.CreateStep("charge-customer",
	func(ctx composer.StepExecutionContext, input WorkflowInput) (composer.StepResponse[string], error) {
		// TODO charge the customer
		return composer.NewStepResponse("charged"), nil
	}, nil)

// region: workflow
// ChargeCustomerWorkflow locks the order while it charges the customer.
var ChargeCustomerWorkflow = composer.CreateWorkflow("charge-customer",
	func(e *composer.Execution, input WorkflowInput) (string, error) {
		_, err := lockingsteps.AcquireLockStep.Run(e, lockingsteps.AcquireLockStepInput{
			Key: []string{input.OrderID},
			// Attempt to acquire the lock for two seconds before timing out
			Timeout: 2,
			// Lock is only held for a maximum of ten seconds
			TTL: 10,
		})
		if err != nil {
			return "", err
		}

		result, err := chargeCustomerStep.Run(e, input)
		if err != nil {
			return "", err
		}

		_, err = lockingsteps.ReleaseLockStep.Run(e, lockingsteps.ReleaseLockStepInput{
			Key: []string{input.OrderID},
		})
		return result, err
	})

// endregion: workflow
