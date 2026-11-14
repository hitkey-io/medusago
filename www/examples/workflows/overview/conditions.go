package overview

import "github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"

// ConditionalInput is the input of the conditional workflow.
type ConditionalInput struct {
	Name  string `json:"name"`
	Greet bool   `json:"greet"`
}

// region: conditions
// ConditionalWorkflow only runs step2 if the input says so. The body is
// regular Go code, so a condition is an `if`.
var ConditionalWorkflow = composer.CreateWorkflow("conditional-hello",
	func(e *composer.Execution, input ConditionalInput) (string, error) {
		if !input.Greet {
			return "", nil
		}

		return step2.Run(e, WorkflowInput{Name: input.Name})
	})

// endregion: conditions

// region: when
// NamedConditionWorkflow records the condition as a step named
// "when-greet" in the execution's journal. step2 only runs if the condition
// holds.
var NamedConditionWorkflow = composer.CreateWorkflow("named-condition-hello",
	func(e *composer.Execution, input ConditionalInput) (string, error) {
		return composer.When(e, "when-greet", input.Greet, func() (string, error) {
			return step2.Run(e, WorkflowInput{Name: input.Name})
		})
	})

// endregion: when
