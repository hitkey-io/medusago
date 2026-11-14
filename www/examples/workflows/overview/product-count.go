package overview

import (
	"context"
	"errors"

	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// caller is what the services of the modules generated with MedusaService
// offer: their methods called by name.
type caller interface {
	Call(ctx context.Context, method string, args []any) (any, error)
}

// ProductCountOutput is what the productCount workflow returns.
type ProductCountOutput struct {
	Count int `json:"count"`
}

// region: container
// getProductCountStep resolves the Product Module's service from the
// container, which is in the step's context.
var getProductCountStep = composer.CreateStep("get-product-count",
	func(ctx composer.StepExecutionContext, _ any) (composer.StepResponse[int], error) {
		productModuleService, ok := ctx.Container.Resolve(utilsmodulessdk.PRODUCT).(caller)
		if !ok {
			return composer.StepResponse[int]{}, errors.New("the Product Module isn't registered")
		}

		// listAndCountProducts returns the products and their total count
		result, err := productModuleService.Call(ctx.Context, "listAndCountProducts", nil)
		if err != nil {
			return composer.StepResponse[int]{}, err
		}
		count := result.([]any)[1].(int)

		return composer.NewStepResponse(count), nil
	}, nil)

// ProductCountWorkflow returns the total number of products.
var ProductCountWorkflow = composer.CreateWorkflow("product-count",
	func(e *composer.Execution, _ any) (ProductCountOutput, error) {
		count, err := getProductCountStep.Run(e, nil)
		if err != nil {
			return ProductCountOutput{}, err
		}

		return ProductCountOutput{Count: count}, nil
	})

// endregion: container
