package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CalculateShippingOptionsPricesStep mirrors calculateShippingOptionsPricesStep.
var CalculateShippingOptionsPricesStep = composer.CreateStep("calculate-shipping-options-prices",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		prices, err := Fulfillment(ctx.Container).CalculateShippingOptionsPrices(ctx.Context, input)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(prices), nil
	}, nil)
