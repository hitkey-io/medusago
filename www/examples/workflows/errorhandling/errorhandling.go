// Package errorhandling holds the examples of the Error Handling in Workflows
// chapter.
package errorhandling

import (
	"errors"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// actionThatReturnsError stands for a step that fails.
var actionThatReturnsError = composer.CreateStep("action-that-returns-error",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		return composer.StepResponse[string]{}, common.NewMedusaError(
			common.MedusaErrorTypes.UNEXPECTED_STATE,
			"Oops! Something happened.",
		)
	}, nil)

// moreActions stands for a step that follows it.
var moreActions = composer.CreateStep("more-actions",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		return composer.NewStepResponse("more actions done"), nil
	}, nil)

// region: workflow
// MyWorkflow runs the steps one after the other. If actionThatReturnsError
// fails, the workflow stops and returns its error.
var MyWorkflow = composer.CreateWorkflow("my-workflow",
	func(e *composer.Execution, _ any) (string, error) {
		if _, err := actionThatReturnsError.Run(e, nil); err != nil {
			return "", err
		}

		return moreActions.Run(e, nil)
	})

// endregion: workflow

// region: route
// GET runs the workflow and handles its error.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	result, err := MyWorkflow.Run(req.Ctx(), req.Scope, nil)
	if err != nil {
		// the error is the one the step returned, unchanged
		var medusaErr *common.MedusaError
		if errors.As(err, &medusaErr) && medusaErr.Type == common.MedusaErrorTypes.UNEXPECTED_STATE {
			return res.Status(500).JSON(map[string]any{
				"message": "Something unexpected happened. Please try again.",
			})
		}

		// any other error is handled by Medusa's error handler
		return err
	}

	return res.JSON(map[string]any{"result": result})
}

// endregion: route

// region: handle
// SkipWorkflow handles the error of a step in the workflow's body, and
// doesn't run the rest of the steps. The workflow then succeeds: it is the
// equivalent of a try-catch block that doesn't do anything in its catch
// block, so the steps that ran aren't compensated.
var SkipWorkflow = composer.CreateWorkflow("skip-on-failure",
	func(e *composer.Execution, _ any) (string, error) {
		if _, err := actionThatReturnsError.Run(e, nil); err != nil {
			return "skipped", nil
		}

		// this step isn't executed if the previous step fails
		return moreActions.Run(e, nil)
	})

// endregion: handle

// region: continue
// continueStep is actionThatReturnsError configured to let the workflow
// continue if it fails permanently.
var continueStep = actionThatReturnsError.Config("", composer.StepOptions{
	ContinueOnPermanentFailure: true,
})

// ContinueWorkflow runs moreActions even if continueStep fails.
var ContinueWorkflow = composer.CreateWorkflow("continue-on-failure",
	func(e *composer.Execution, _ any) (string, error) {
		// if the step fails, its output is the zero value and err is nil
		if _, err := continueStep.Run(e, nil); err != nil {
			return "", err
		}

		// this step is executed even if the previous step fails
		return moreActions.Run(e, nil)
	})

// endregion: continue

// region: permanent
// permanentFailureStep fails permanently, so it isn't retried even if it has
// retries left.
var permanentFailureStep = composer.CreateStep("permanent-failure-step",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		return composer.PermanentFailure[string]("The payment provider rejected the request"), nil
	}, nil).Config("", composer.StepOptions{MaxRetries: 3})

// endregion: permanent
