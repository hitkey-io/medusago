// Package workflows mirrors @medusajs/core-flows/src/shipping-profile/workflows.
package workflows

import (
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// validateStepShippingProfileDelete mirrors validateStepShippingProfileDelete.
var validateStepShippingProfileDelete = composer.CreateStep("validate-step-shipping-profile-delete",
	func(ctx composer.StepExecutionContext, links []any) (composer.StepResponse[any], error) {
		if len(links) > 0 {
			ids := make([]string, len(links))
			for i, l := range links {
				r, _ := l.(*dal.Record)
				v, _ := r.Get("product_id")
				ids[i] = fmt.Sprint(v)
			}
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Cannot delete following shipping profiles because they are linked to products: "+strings.Join(ids, ", "))
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// DeleteShippingProfileWorkflow mirrors deleteShippingProfileWorkflow.
var DeleteShippingProfileWorkflow = composer.CreateWorkflow("delete-shipping-profile-workflow",
	func(e *composer.Execution, ids []any) (any, error) {
		f := dal.NewRecord()
		f.Set("shipping_profile_id", ids)
		q, err := commonsteps.UseQueryGraphStep.Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "product_shipping_profile", Fields: []string{"product_id", "shipping_profile_id"}, Filters: f},
		})
		if err != nil {
			return nil, err
		}
		links, _ := q.Data.([]any)
		if _, err := validateStepShippingProfileDelete.Run(e, links); err != nil {
			return nil, err
		}
		if _, err := fulfillmentsteps.DeleteShippingProfilesStep.Run(e, ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("shipping_profile_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.FULFILLMENT, keys)
		_, err = commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry})
		return nil, err
	})
