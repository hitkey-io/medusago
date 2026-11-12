package workflows

import (
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/store-credit/steps"
)

// ValidateStoreCreditAccountInputStep mirrors
// validateStoreCreditAccountInputStep of create-store-credit-accounts.ts.
var ValidateStoreCreditAccountInputStep = composer.CreateStep("validate-store-credit-account-input",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[any], error) {
		if len(input) == 0 {
			return composer.StepResponse[any]{}, invalidData("No input provided")
		}
		for _, i := range input {
			if !cartutils.IsPresent(get(i, "currency_code")) {
				return composer.StepResponse[any]{}, invalidData("Currency code is required to create a store credit account")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateStoreCreditAccountsWorkflow mirrors createStoreCreditAccountsWorkflow
// ("create-store-credit-accounts"): records { code?, customer_id,
// currency_code, metadata? }.
var CreateStoreCreditAccountsWorkflow = composer.CreateWorkflow("create-store-credit-accounts",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		if _, err := ValidateStoreCreditAccountInputStep.Run(e, input); err != nil {
			return nil, err
		}
		return steps.CreateStoreCreditAccountsStep.Run(e, input)
	})
