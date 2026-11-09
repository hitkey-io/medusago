// Package workflows mirrors @medusajs/core-flows/src/fulfillment/workflows.
package workflows

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateServiceZonesWorkflow mirrors createServiceZonesWorkflow.
var CreateServiceZonesWorkflow = composer.CreateWorkflow("create-service-zones-workflow",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateServiceZonesStep.Run(e, data)
	})

// UpdateServiceZonesWorkflow mirrors updateServiceZonesWorkflow.
var UpdateServiceZonesWorkflow = composer.CreateWorkflow("update-service-zones-workflow",
	func(e *composer.Execution, in steps.SelectorUpdateInput) ([]*dal.Record, error) {
		return steps.UpdateServiceZonesStep.Run(e, in)
	})

// DeleteServiceZonesWorkflow mirrors deleteServiceZonesWorkflow.
var DeleteServiceZonesWorkflow = composer.CreateWorkflow("delete-service-zones-workflow",
	func(e *composer.Execution, ids []any) (any, error) {
		return steps.DeleteServiceZonesStep.Run(e, ids)
	})

// DeleteFulfillmentSetsWorkflow mirrors deleteFulfillmentSetsWorkflow.
var DeleteFulfillmentSetsWorkflow = composer.CreateWorkflow("delete-fulfillment-sets-workflow",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteFulfillmentSetsStep.Run(e, ids); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("fulfillment_set_id", ids)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.FULFILLMENT, keys)
		_, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry})
		return nil, err
	})

// CreateShippingProfilesWorkflow mirrors createShippingProfilesWorkflow.
var CreateShippingProfilesWorkflow = composer.CreateWorkflow("create-shipping-profiles-workflow",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateShippingProfilesStep.Run(e, data)
	})

// UpdateShippingProfilesWorkflow mirrors updateShippingProfilesWorkflow.
var UpdateShippingProfilesWorkflow = composer.CreateWorkflow("update-shipping-profiles-workflow",
	func(e *composer.Execution, in steps.SelectorUpdateInput) ([]*dal.Record, error) {
		return steps.UpdateShippingProfilesStep.Run(e, in)
	})
