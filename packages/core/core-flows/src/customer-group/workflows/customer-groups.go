// Package workflows mirrors @medusajs/core-flows/src/customer-group/workflows.
package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer-group/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateCustomerGroupsInput mirrors CreateCustomerGroupsWorkflowInput.
type CreateCustomerGroupsInput struct {
	CustomersData []*dal.Record `json:"customersData"`
}

// DeleteCustomerGroupsInput mirrors DeleteCustomerGroupsWorkflowInput.
type DeleteCustomerGroupsInput struct {
	IDs []any `json:"ids"`
}

// CreateCustomerGroupsWorkflow mirrors createCustomerGroupsWorkflow.
var CreateCustomerGroupsWorkflow = composer.CreateWorkflow("create-customer-groups",
	func(e *composer.Execution, in CreateCustomerGroupsInput) ([]*dal.Record, error) {
		return steps.CreateCustomerGroupsStep.Run(e, in.CustomersData)
	})

// UpdateCustomerGroupsWorkflow mirrors updateCustomerGroupsWorkflow.
var UpdateCustomerGroupsWorkflow = composer.CreateWorkflow("update-customer-groups",
	func(e *composer.Execution, in steps.UpdateCustomerGroupsStepInput) ([]*dal.Record, error) {
		return steps.UpdateCustomerGroupsStep.Run(e, in)
	})

// DeleteCustomerGroupsWorkflow mirrors deleteCustomerGroupsWorkflow.
var DeleteCustomerGroupsWorkflow = composer.CreateWorkflow("delete-customer-groups",
	func(e *composer.Execution, in DeleteCustomerGroupsInput) (any, error) {
		return steps.DeleteCustomerGroupStep.Run(e, in.IDs)
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
