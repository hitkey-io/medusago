package locks

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CustomCompleteCartInput is the input of customCompleteCartWorkflow.
type CustomCompleteCartInput struct {
	CartID string `json:"cart_id"`
}

// region: nested
// CustomCompleteCartWorkflow acquires the lock of the cart around the
// nested workflow, as the lock steps of a workflow run as a step don't
// lock.
var CustomCompleteCartWorkflow = composer.CreateWorkflow("custom-complete-cart",
	func(e *composer.Execution, input CustomCompleteCartInput) (*dal.Record, error) {
		// acquire the same lock as the nested workflow
		_, err := lockingsteps.AcquireLockStep.Run(e, lockingsteps.AcquireLockStepInput{
			Key:     []string{input.CartID},
			Timeout: 30,
			TTL:     60 * 2,
		})
		if err != nil {
			return nil, err
		}

		// execute the nested workflow
		result, err := workflows.CompleteCartWorkflow.RunAsStep(e, workflows.CompleteCartWorkflowInput{ID: input.CartID})
		if err != nil {
			return nil, err
		}

		// release the lock after the nested workflow is complete
		_, err = lockingsteps.ReleaseLockStep.Run(e, lockingsteps.ReleaseLockStepInput{
			Key: []string{input.CartID},
		})
		return result, err
	})

// endregion: nested
