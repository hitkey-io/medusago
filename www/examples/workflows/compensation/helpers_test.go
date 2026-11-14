package compensation

import (
	"errors"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

var deleteWorkflow = composer.CreateWorkflow("delete-items", func(e *composer.Execution, in DeleteItemsInput) ([]string, error) {
	return deleteItemsStep.Run(e, in)
})

var deleteThenFailWorkflow = composer.CreateWorkflow("delete-items-then-fail", func(e *composer.Execution, in DeleteItemsInput) ([]string, error) {
	out, err := deleteItemsStep.Run(e, in)
	if err != nil {
		return nil, err
	}
	failing := composer.CreateStep("fail", func(composer.StepExecutionContext, any) (composer.StepResponse[any], error) {
		return composer.StepResponse[any]{}, errors.New("later failure")
	}, nil)
	_, err = failing.Run(e, nil)
	return out, err
})
