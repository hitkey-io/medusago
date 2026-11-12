package workflows

import (
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type receiveCompleteReturnValidation struct {
	OrderReturn *dal.Record `json:"orderReturn"`
	Input       *dal.Record `json:"input"`
}

var receiveCompleteReturnValidationStep = composer.CreateStep("receive-return-order-validation",
	func(ctx composer.StepExecutionContext, in receiveCompleteReturnValidation) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfItemsDoesNotExistsInReturn(in.OrderReturn, recs(g(in.Input, "items"))); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ReceiveAndCompleteReturnOrderWorkflow mirrors
// receiveAndCompleteReturnOrderWorkflow (receive-complete-return.ts): the
// return is read, validated and received in one step. No route runs it.
var ReceiveAndCompleteReturnOrderWorkflow = composer.CreateWorkflow("receive-return-order",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		orderReturn, err := remoteOne(e, "", "returns", idVars(g(input, "return_id")), []string{"id", "canceled_at", "items.*"}, true)
		if err != nil {
			return nil, err
		}
		if _, err := receiveCompleteReturnValidationStep.Run(e, receiveCompleteReturnValidation{OrderReturn: orderReturn, Input: input}); err != nil {
			return nil, err
		}
		return ordersteps.ReceiveReturnStep.Run(e, input)
	})
