package retry

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: waiting
// waitForImportStep checks the status of an import job in a third-party
// service. While the job isn't completed, it returns an empty response, so
// the step stays waiting.
var waitForImportStep = composer.CreateStep("wait-for-import",
	func(ctx composer.StepExecutionContext, jobID string) (composer.StepResponse[any], error) {
		job := getImportJob(jobID, ctx.Attempt)
		if job != "completed" {
			return composer.StepResponse[any]{}, nil
		}

		return composer.NewStepResponse[any]("import completed"), nil
	}, nil).Config("", composer.StepOptions{Async: true})

// ImportWorkflow waits for an import job.
var ImportWorkflow = composer.CreateWorkflow("import-job",
	func(e *composer.Execution, jobID string) (any, error) {
		return waitForImportStep.Run(e, jobID)
	})

// endregion: waiting

// getImportJob stands for a call to the third-party service.
func getImportJob(jobID string, attempt int) string {
	if attempt < 2 {
		return "running"
	}
	return "completed"
}

// region: retry-step
// RetryWaitForImport executes the handler of the waiting step again.
func RetryWaitForImport(ctx context.Context, container composer.Container, transactionID string) error {
	_, err := composer.RetryStep(ctx, container, composer.IdempotencyKey{
		Action:        "invoke",
		TransactionID: transactionID,
		StepID:        "wait-for-import",
		WorkflowID:    "import-job",
	})
	return err
}

// endregion: retry-step
