// Package timeout holds the examples of the Workflow Timeout chapter.
package timeout

import (
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: step
// waitStep is an async step that waits for its status to be set from
// outside. It fails if that takes longer than two seconds.
var waitStep = composer.CreateStep("wait-step",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[any], error) {
		// an empty response: the step waits
		return composer.StepResponse[any]{}, nil
	}, nil).Config("", composer.StepOptions{
	Async:   true,
	Timeout: 2, // seconds
})

// MyWorkflow runs waitStep.
var MyWorkflow = composer.CreateWorkflow("wait-workflow",
	func(e *composer.Execution, _ any) (any, error) {
		return waitStep.Run(e, nil)
	})

// endregion: step
