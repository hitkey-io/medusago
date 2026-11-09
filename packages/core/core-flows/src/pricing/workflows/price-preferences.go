// Package workflows mirrors @medusajs/core-flows/src/pricing/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/pricing/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
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
