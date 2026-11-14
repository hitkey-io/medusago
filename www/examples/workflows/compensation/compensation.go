// Package compensation holds the examples of the Compensation Function
// chapter.
package compensation

import (
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: step1
// step1 has a compensation function, the third argument of CreateStep. It
// only runs if an error occurs throughout the workflow.
var step1 = composer.CreateStep("step-1",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		message := "Hello from step one!"
		slog.Info(message)

		return composer.NewStepResponse(message), nil
	},
	func(ctx composer.StepExecutionContext, _ any) error {
		slog.Info("Oops! Rolling back my changes...")
		return nil
	})

// endregion: step1

// region: step2
// step2 fails.
var step2 = composer.CreateStep("step-2",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[any], error) {
		return composer.StepResponse[any]{}, common.NewMedusaError(
			common.MedusaErrorTypes.UNEXPECTED_STATE,
			"Throwing an error...",
		)
	}, nil)

// endregion: step2

// region: workflow
// HelloWorldWorkflow runs step1, then step2, which fails. So, the
// compensation function of step1 runs.
var HelloWorldWorkflow = composer.CreateWorkflow("hello-world",
	func(e *composer.Execution, _ any) (string, error) {
		message, err := step1.Run(e, nil)
		if err != nil {
			return "", err
		}
		if _, err := step2.Run(e, nil); err != nil {
			return "", err
		}

		return message, nil
	})

// endregion: workflow

// region: input
// stepWithInput passes data to its compensation function as the second
// argument of NewStepResponse.
var stepWithInput = composer.CreateStep("step-with-input",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		return composer.NewStepResponse(
			"Hello from step one!",
			map[string]string{"message": "Oops! Rolling back my changes..."},
		), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		// data is nil if the step itself failed
		input, ok := data.(map[string]string)
		if !ok {
			return nil
		}

		slog.Info(input["message"])
		return nil
	})

// endregion: input

// region: container
// stepWithContainer resolves resources from the container, which is in the
// context of the compensation function too.
var stepWithContainer = composer.CreateStep("step-with-container",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[string], error) {
		return composer.NewStepResponse(
			"Hello from step one!",
			"Oops! Rolling back my changes...",
		), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		logger := ctx.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)

		message, _ := data.(string)
		logger.Info(message)
		return nil
	})

// endregion: container
