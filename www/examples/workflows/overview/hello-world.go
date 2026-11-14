// Package overview holds the examples of the Workflows chapter.
package overview

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: step1
// step1 is a step with the unique name "step-1". It takes no input.
var step1 = composer.CreateStep("step-1",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		return composer.NewStepResponse("Hello from step one!"), nil
	}, nil)

// endregion: step1

// region: step2
// WorkflowInput is the input of the workflow and of step2.
type WorkflowInput struct {
	Name string `json:"name"`
}

// step2 receives the workflow's input.
var step2 = composer.CreateStep("step-2",
	func(ctx composer.StepExecutionContext, input WorkflowInput) (composer.StepResponse[string], error) {
		return composer.NewStepResponse(fmt.Sprintf("Hello %s from step two!", input.Name)), nil
	}, nil)

// endregion: step2

// region: workflow
// WorkflowOutput is what the workflow returns.
type WorkflowOutput struct {
	Message string `json:"message"`
}

// HelloWorldWorkflow is the workflow with the unique name "hello-world".
var HelloWorldWorkflow = composer.CreateWorkflow("hello-world",
	func(e *composer.Execution, input WorkflowInput) (WorkflowOutput, error) {
		if _, err := step1.Run(e, nil); err != nil {
			return WorkflowOutput{}, err
		}
		// to pass input
		message, err := step2.Run(e, input)
		if err != nil {
			return WorkflowOutput{}, err
		}

		return WorkflowOutput{Message: message}, nil
	})

// endregion: workflow
