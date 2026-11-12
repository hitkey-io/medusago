package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateCartCreditLinesWorkflow mirrors createCartCreditLinesWorkflow
// (create-cart-credit-lines.ts): no route runs it.
var CreateCartCreditLinesWorkflow = composer.CreateWorkflow("create-cart-credit-lines",
	func(e *composer.Execution, input []any) ([]any, error) {
		return commonsteps.CreateEntitiesStep.Run(e, commonsteps.EntitiesStepInput{
			ModuleRegistrationName: utilsmodulessdk.CART,
			InvokeMethod:           "createCreditLines",
			CompensateMethod:       "deleteCreditLines",
			Data:                   input,
		})
	})

// DeleteCartCreditLinesInput mirrors the input of deleteCartCreditLinesWorkflow.
type DeleteCartCreditLinesInput struct {
	ID []any `json:"id"`
}

// DeleteCartCreditLinesWorkflow mirrors deleteCartCreditLinesWorkflow
// (delete-cart-credit-lines.ts): no route runs it.
var DeleteCartCreditLinesWorkflow = composer.CreateWorkflow("delete-cart-credit-lines",
	func(e *composer.Execution, input DeleteCartCreditLinesInput) (any, error) {
		_, err := commonsteps.DeleteEntitiesStep.Run(e, commonsteps.EntitiesStepInput{
			ModuleRegistrationName: utilsmodulessdk.CART,
			InvokeMethod:           "softDeleteCreditLines",
			CompensateMethod:       "restoreCreditLines",
			Data:                   input.ID,
		})
		return nil, err
	})
