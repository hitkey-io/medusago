// Package store holds the examples of the Store Workflow Executions
// chapter.
package store

import (
	"log/slog"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var step1 = composer.CreateStep("step-1",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[any], error) {
		slog.Info("Hello from step 1")
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// region: workflow
// HelloWorkflow keeps its executions stored in the database for 99,999
// seconds after they finish.
var HelloWorkflow = composer.CreateWorkflowWithOptions("hello-workflow",
	composer.WorkflowOptions{
		Store:         true,
		RetentionTime: 99999,
	},
	func(e *composer.Execution, _ any) (any, error) {
		return step1.Run(e, nil)
	})

// endregion: workflow
