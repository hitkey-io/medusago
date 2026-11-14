// Package retry holds the examples of the Retry Failed Steps chapter.
package retry

import (
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: step
// step1 is retried twice if it fails: it's executed three times at most.
var step1 = composer.CreateStep("step-1",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		slog.Info("Executing step 1", "attempt", ctx.Attempt)

		return composer.StepResponse[string]{}, common.NewMedusaError(
			common.MedusaErrorTypes.UNEXPECTED_STATE,
			"Oops! Something happened.",
		)
	}, nil).Config("", composer.StepOptions{MaxRetries: 2})

// endregion: step

// MyWorkflow runs step1.
var MyWorkflow = composer.CreateWorkflow("hello-world",
	func(e *composer.Execution, _ any) (string, error) {
		return step1.Run(e, nil)
	})
