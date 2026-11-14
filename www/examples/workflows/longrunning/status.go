package longrunning

import (
	"context"
	"errors"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: success
// SetStep2Success sets the status of step2 of the workflow's execution
// to successful. The workflow then continues with step3.
func SetStep2Success(ctx context.Context, container composer.Container, transactionID string) error {
	return composer.SetStepSuccess(ctx, container, composer.IdempotencyKey{
		Action:        "invoke",
		TransactionID: transactionID,
		StepID:        "step-2",
		WorkflowID:    "hello-world",
	}, "Done!")
}

// endregion: success

// region: failure
// SetStep2Failure sets the status of step2 to failed. The workflow's
// execution fails, and the compensation functions of the steps that ran
// are executed.
func SetStep2Failure(ctx context.Context, container composer.Container, transactionID string) error {
	return composer.SetStepFailure(ctx, container, composer.IdempotencyKey{
		Action:        "invoke",
		TransactionID: transactionID,
		StepID:        "step-2",
		WorkflowID:    "hello-world",
	}, errors.New("Failed!"))
}

// endregion: failure

// region: transaction-id
// RunAndGetTransactionID executes the workflow and returns the ID of its
// transaction.
func RunAndGetTransactionID(ctx context.Context, container composer.Container) (string, error) {
	_, info, err := MyWorkflow.RunWithInfo(ctx, container, nil)
	if err != nil {
		return "", err
	}

	// use info.TransactionID later to change the status of the async steps
	return info.TransactionID, nil
}

// endregion: transaction-id
