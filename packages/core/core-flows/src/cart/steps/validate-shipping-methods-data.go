package steps

import (
	"context"
	"fmt"

	. "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type fulfillmentDataValidator interface {
	ValidateFulfillmentData(ctx context.Context, providerID string, optionData, data, context_ *dal.Record) (any, error)
}

// ValidateAndReturnShippingMethodsDataStep mirrors
// validateAndReturnShippingMethodsDataStep: each option's data validated by
// its provider, as [{ [option id]: data }] (nil without options).
var ValidateAndReturnShippingMethodsDataStep = composer.CreateStep("validate-and-return-shipping-methods-data",
	func(ctx composer.StepExecutionContext, options []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(options) == 0 {
			return composer.NewStepResponse[[]*dal.Record](nil), nil
		}
		svc, _ := ctx.Container.Resolve(utilsmodulessdk.FULFILLMENT).(fulfillmentDataValidator)
		out := make([]*dal.Record, 0, len(options))
		var firstErr error
		for _, o := range options {
			validated, err := svc.ValidateFulfillmentData(ctx.Context, fmt.Sprint(Get(o, "provider_id")), Rec(o, "option_data"), Rec(o, "method_data"), Rec(o, "context"))
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				continue
			}
			r := dal.NewRecord()
			r.Set(fmt.Sprint(Get(o, "id")), validated)
			out = append(out, r)
		}
		if firstErr != nil {
			return composer.StepResponse[[]*dal.Record]{}, firstErr
		}
		return composer.NewStepResponse(out), nil
	}, nil)
