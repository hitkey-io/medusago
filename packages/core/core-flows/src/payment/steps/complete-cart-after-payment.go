package steps

import (
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CompleteCartWorkflowID mirrors completeCartWorkflowId (the cart workflows
// import this package, so the workflow is run by its id).
const CompleteCartWorkflowID = "complete-cart"

// CompleteCartAfterPaymentStepInput mirrors CompleteCartAfterPaymentStepInput.
type CompleteCartAfterPaymentStepInput struct {
	CartID string `json:"cart_id"`
}

// CompleteCartAfterPaymentStep mirrors completeCartAfterPaymentStep: the
// cart is completed through the workflow engine.
var CompleteCartAfterPaymentStep = composer.CreateStep("complete-cart-after-payment-step",
	func(ctx composer.StepExecutionContext, in CompleteCartAfterPaymentStepInput) (composer.StepResponse[any], error) {
		if _, _, err := composer.RunByID(ctx.Context, ctx.Container, CompleteCartWorkflowID, in.CartID, composer.RunOptions{}); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)
