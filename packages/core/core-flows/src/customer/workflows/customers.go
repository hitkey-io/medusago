// Package workflows mirrors @medusajs/core-flows/src/customer/workflows.
package workflows

import (
	"fmt"
	authsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/steps"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(ids []any) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		ev := dal.NewRecord()
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

func hook(e *composer.Execution, name string, kv ...any) error {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return composer.CreateHook(e, name, r)
}

// CreateCustomersInput mirrors CreateCustomersWorkflowInput.
type CreateCustomersInput struct {
	CustomersData  []*dal.Record
	AdditionalData *dal.Record
}

// CreateCustomersWorkflow mirrors createCustomersWorkflow.
var CreateCustomersWorkflow = composer.CreateWorkflow("create-customers",
	func(e *composer.Execution, input CreateCustomersInput) ([]*dal.Record, error) {
		created, err := steps.CreateCustomersStep.Run(e, input.CustomersData)
		if err != nil {
			return nil, err
		}
		if err := hook(e, "customersCreated", "customers", created, "additional_data", input.AdditionalData); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "customer.created", Data: idEvents(commonsteps.RecordIDs(created))}); err != nil {
			return nil, err
		}
		return created, nil
	})

// UpdateCustomersInput mirrors UpdateCustomersWorkflowInput.
type UpdateCustomersInput struct {
	Selector       *dal.Record
	Update         *dal.Record
	AdditionalData *dal.Record
}

// UpdateCustomersWorkflow mirrors updateCustomersWorkflow.
var UpdateCustomersWorkflow = composer.CreateWorkflow("update-customers",
	func(e *composer.Execution, input UpdateCustomersInput) ([]*dal.Record, error) {
		updated, err := steps.UpdateCustomersStep.Run(e, steps.SelectorUpdateInput{Selector: input.Selector, Update: input.Update})
		if err != nil {
			return nil, err
		}
		if err := hook(e, "customersUpdated", "customers", updated, "additional_data", input.AdditionalData); err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "customer.updated", Data: idEvents(commonsteps.RecordIDs(updated))}); err != nil {
			return nil, err
		}
		return updated, nil
	})

// DeleteCustomersWorkflow mirrors deleteCustomersWorkflow.
var DeleteCustomersWorkflow = composer.CreateWorkflow("delete-customers",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteCustomersStep.Run(e, ids); err != nil {
			return nil, err
		}
		if err := hook(e, "customersDeleted", "ids", ids); err != nil {
			return nil, err
		}
		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "customer.deleted", Data: idEvents(ids)})
		return nil, err
	})

// CreateCustomerAddressesInput mirrors CreateCustomerAddressesWorkflowInput.
type CreateCustomerAddressesInput struct {
	Addresses      []*dal.Record
	AdditionalData *dal.Record
}

// CreateCustomerAddressesWorkflow mirrors createCustomerAddressesWorkflow.
var CreateCustomerAddressesWorkflow = composer.CreateWorkflow("create-customer-addresses",
	func(e *composer.Execution, input CreateCustomerAddressesInput) ([]*dal.Record, error) {
		unset := steps.MaybeUnsetInput{Create: input.Addresses}
		if unset.Create == nil {
			unset.Create = []*dal.Record{}
		}
		err := composer.Parallelize(
			func() error { _, err := steps.MaybeUnsetDefaultShippingAddressesStep.Run(e, unset); return err },
			func() error { _, err := steps.MaybeUnsetDefaultBillingAddressesStep.Run(e, unset); return err },
		)
		if err != nil {
			return nil, err
		}
		created, err := steps.CreateCustomerAddressesStep.Run(e, input.Addresses)
		if err != nil {
			return nil, err
		}
		if err := hook(e, "addressesCreated", "addresses", created, "additional_data", input.AdditionalData); err != nil {
			return nil, err
		}
		return created, nil
	})

// UpdateCustomerAddressesWorkflow mirrors updateCustomerAddressesWorkflow.
var UpdateCustomerAddressesWorkflow = composer.CreateWorkflow("update-customer-addresses",
	func(e *composer.Execution, input UpdateCustomersInput) ([]*dal.Record, error) {
		su := steps.SelectorUpdateInput{Selector: input.Selector, Update: input.Update}
		unset := steps.MaybeUnsetInput{Update: &su}
		err := composer.Parallelize(
			func() error { _, err := steps.MaybeUnsetDefaultShippingAddressesStep.Run(e, unset); return err },
			func() error { _, err := steps.MaybeUnsetDefaultBillingAddressesStep.Run(e, unset); return err },
		)
		if err != nil {
			return nil, err
		}
		updated, err := steps.UpdateCustomerAddressesStep.Run(e, su)
		if err != nil {
			return nil, err
		}
		if err := hook(e, "addressesUpdated", "addresses", updated, "additional_data", input.AdditionalData); err != nil {
			return nil, err
		}
		return updated, nil
	})

// DeleteCustomerAddressesWorkflow mirrors deleteCustomerAddressesWorkflow.
var DeleteCustomerAddressesWorkflow = composer.CreateWorkflow("delete-customer-addresses",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteCustomerAddressesStep.Run(e, ids); err != nil {
			return nil, err
		}
		return nil, hook(e, "addressesDeleted", "ids", ids)
	})

// getCustomerAuthIdentityStep mirrors getCustomerAuthIdentityStep.
var getCustomerAuthIdentityStep = composer.CreateStep("get-customer-auth-identity",
	func(ctx composer.StepExecutionContext, identities []*dal.Record) (composer.StepResponse[*dal.Record], error) {
		if len(identities) == 0 {
			return composer.StepResponse[*dal.Record]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "Auth identity not found")
		}
		return composer.NewStepResponse(identities[0]), nil
	}, nil)

// RemoveCustomerAccountWorkflow mirrors removeCustomerAccountWorkflow.
var RemoveCustomerAccountWorkflow = composer.CreateWorkflow("remove-customer-account",
	func(e *composer.Execution, customerID string) (string, error) {
		f := dal.NewRecord()
		f.Set("id", customerID)
		isList := false
		q, err := commonsteps.UseQueryGraphStep.Config("get-customer", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: graphInput("customer", []string{"id", "has_account"}, f), IsList: &isList,
		})
		if err != nil {
			return "", err
		}
		if _, err := DeleteCustomersWorkflow.RunAsStep(e, []any{customerID}); err != nil {
			return "", err
		}
		customer, _ := q.Data.(*dal.Record)
		hasAccount := false
		if customer != nil {
			v, _ := customer.Get("has_account")
			hasAccount, _ = v.(bool)
		}
		if !hasAccount {
			return customerID, nil
		}
		af := dal.NewRecord()
		meta := dal.NewRecord()
		meta.Set("customer_id", customerID)
		af.Set("app_metadata", meta)
		ids, err := commonsteps.UseQueryGraphStep.Config("get-auth-identities", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: graphInput("auth_identity", []string{"id", "app_metadata", "provider_identities.entity_id"}, af),
		})
		if err != nil {
			return "", err
		}
		rows, _ := ids.Data.([]any)
		var identities []*dal.Record
		for _, r := range rows {
			if rec, ok := r.(*dal.Record); ok {
				identities = append(identities, rec)
			}
		}
		identity, err := getCustomerAuthIdentityStep.Run(e, identities)
		if err != nil {
			return "", err
		}
		keep := false
		if m, ok := identity.Get("app_metadata"); ok {
			if mr, ok := m.(*dal.Record); ok {
				for _, k := range mr.Keys() {
					if k == "customer_id" {
						continue
					}
					if v, _ := mr.Get(k); v != nil {
						keep = true
					}
				}
			}
		}
		id, _ := identity.Get("id")
		if keep {
			_, err = authsteps.SetAuthAppMetadataStep.Run(e, authsteps.SetAuthAppMetadataStepInput{AuthIdentityID: id, ActorType: "customer", Value: nil})
		} else {
			_, err = authsteps.DeleteAuthIdentityStep.Run(e, []any{id})
		}
		if err != nil {
			return "", fmt.Errorf("%w", err)
		}
		return customerID, nil
	})
