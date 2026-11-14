package longrunning

import (
	"time"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ImportOutput is what the import step returns.
type ImportOutput struct {
	Imported int `json:"imported"`
}

// region: background
// importStep takes too long to wait for. With BackgroundExecution, the
// workflow's run returns to its caller as soon as the step starts, and the
// step's handler continues in the background. Its response completes the
// step, unlike the response of a step that is only async.
var importStep = composer.CreateStep("import-step",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[ImportOutput], error) {
		// ... import the data, which takes long
		time.Sleep(10 * time.Millisecond)

		return composer.NewStepResponse(ImportOutput{Imported: 1000}), nil
	}, nil).Config("", composer.StepOptions{Async: true, BackgroundExecution: true})

// ImportWorkflow imports data in the background.
var ImportWorkflow = composer.CreateWorkflow("import-data",
	func(e *composer.Execution, _ any) (ImportOutput, error) {
		return importStep.Run(e, nil)
	})

// endregion: background
