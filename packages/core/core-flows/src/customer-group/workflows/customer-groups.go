// Package workflows mirrors @medusajs/core-flows/src/customer-group/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateCustomerGroupsWorkflow mirrors createCustomerGroupsWorkflow.
var CreateCustomerGroupsWorkflow = composer.CreateWorkflow("create-customer-groups",
	func(e *composer.Execution, data []*dal.Record) ([]*dal.Record, error) {
		return steps.CreateCustomerGroupsStep.Run(e, data)
	})

// UpdateCustomerGroupsWorkflow mirrors updateCustomerGroupsWorkflow.
var UpdateCustomerGroupsWorkflow = composer.CreateWorkflow("update-customer-groups",
	func(e *composer.Execution, in steps.UpdateCustomerGroupsStepInput) ([]*dal.Record, error) {
		return steps.UpdateCustomerGroupsStep.Run(e, in)
	})

// DeleteCustomerGroupsWorkflow mirrors deleteCustomerGroupsWorkflow.
var DeleteCustomerGroupsWorkflow = composer.CreateWorkflow("delete-customer-groups",
	func(e *composer.Execution, ids []any) (any, error) {
		return steps.DeleteCustomerGroupStep.Run(e, ids)
	})

// LinkCustomersToCustomerGroupWorkflow mirrors linkCustomersToCustomerGroupWorkflow.
var LinkCustomersToCustomerGroupWorkflow = composer.CreateWorkflow("link-customers-to-customer-group",
	func(e *composer.Execution, in steps.LinkInput) (any, error) {
		return steps.LinkCustomersToCustomerGroupStep.Run(e, in)
	})

// LinkCustomerGroupsToCustomerWorkflow mirrors linkCustomerGroupsToCustomerWorkflow.
var LinkCustomerGroupsToCustomerWorkflow = composer.CreateWorkflow("link-customer-groups-to-customer",
	func(e *composer.Execution, in steps.LinkInput) (any, error) {
		return steps.LinkCustomerGroupsToCustomerStep.Run(e, in)
	})
