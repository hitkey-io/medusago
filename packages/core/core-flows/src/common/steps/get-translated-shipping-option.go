package steps

import (
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// GetTranslatedShippingOptionsStepInput mirrors
// GetTranslatedShippingOptionsStepInput.
type GetTranslatedShippingOptionsStepInput struct {
	ShippingOptions []*dal.Record
	Locale          string
}

// GetTranslatedShippingOptionsStep mirrors getTranslatedShippingOptionsStep:
// applyTranslations over the shipping options (in place).
var GetTranslatedShippingOptionsStep = composer.CreateStep("get-translated-shipping-options",
	func(ctx composer.StepExecutionContext, in GetTranslatedShippingOptionsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		q, _ := ctx.Container.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
		objects := make([]any, len(in.ShippingOptions))
		for i, o := range in.ShippingOptions {
			objects[i] = o
		}
		if err := q.ApplyTranslations(ctx.Context, in.Locale, objects); err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(in.ShippingOptions), nil
	}, nil)
