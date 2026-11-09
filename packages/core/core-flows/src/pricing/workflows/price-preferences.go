// Package workflows mirrors @medusajs/core-flows/src/pricing/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreatePricePreferencesWorkflow mirrors createPricePreferencesWorkflow.
var CreatePricePreferencesWorkflow = composer.CreateWorkflow("create-price-preferences",
	func(e *composer.Execution, input []*dal.Record) ([]*dal.Record, error) {
		return steps.CreatePricePreferencesStep.Run(e, input)
	})

// UpdatePricePreferencesWorkflow mirrors updatePricePreferencesWorkflow.
var UpdatePricePreferencesWorkflow = composer.CreateWorkflow("update-price-preferences",
	func(e *composer.Execution, input steps.UpdatePricePreferencesInput) ([]*dal.Record, error) {
		return steps.UpdatePricePreferencesStep.Run(e, input)
	})

// DeletePricePreferencesWorkflow mirrors deletePricePreferencesWorkflow.
var DeletePricePreferencesWorkflow = composer.CreateWorkflow("delete-price-preferences",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeletePricePreferencesStep.Run(e, ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("price_preference_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.PRICING, keys)
		if _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}
		return nil, nil
	})
