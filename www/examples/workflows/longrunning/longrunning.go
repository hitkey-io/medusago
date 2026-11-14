// Package longrunning holds the examples of the Long-Running Workflows
// chapter.
package longrunning

import (
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: steps
// step1 is a regular step.
var step1 = composer.CreateStep("step-1",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[any], error) {
		return composer.NewStepResponse[any](map[string]any{}), nil
	}, nil)

// step2 is an async step: its handler returns an empty response, so the
// step waits for its status to be set from outside.
var step2 = composer.CreateStep("step-2",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[any], error) {
		slog.Info("Waiting to be successful...")

		return composer.StepResponse[any]{}, nil
	}, nil).Config("", composer.StepOptions{Async: true})

// step3 runs once step2 succeeds.
var step3 = composer.CreateStep("step-3",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		return composer.NewStepResponse("Finished three steps"), nil
	}, nil)

// endregion: steps

// region: workflow
// WorkflowOutput is what the workflow returns.
type WorkflowOutput struct {
	Message string `json:"message"`
}

// MyWorkflow is a long-running workflow, since step2 is async and
// returns no response.
var MyWorkflow = composer.CreateWorkflow("hello-world",
	func(e *composer.Execution, _ any) (WorkflowOutput, error) {
		if _, err := step1.Run(e, nil); err != nil {
			return WorkflowOutput{}, err
		}
		if _, err := step2.Run(e, nil); err != nil {
			return WorkflowOutput{}, err
		}
		message, err := step3.Run(e, nil)
		if err != nil {
			return WorkflowOutput{}, err
		}

		return WorkflowOutput{Message: message}, nil
	})

// endregion: workflow
