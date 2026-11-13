// Package steps mirrors @medusajs/core-flows/src/customer-group/steps.
package steps

import (
	"context"
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func customers(c composer.Container) commonsteps.ModuleService {
	return commonsteps.ModuleOf(c, utilsmodulessdk.CUSTOMER)
}

// CreateCustomerGroupsStep mirrors createCustomerGroupsStep.
var CreateCustomerGroupsStep = commonsteps.CreateRecordsStep("create-customer-groups", utilsmodulessdk.CUSTOMER, "CustomerGroup")

// DeleteCustomerGroupStep mirrors deleteCustomerGroupStep.
var DeleteCustomerGroupStep = commonsteps.SoftDeleteRecordsStep("delete-customer-groups", utilsmodulessdk.CUSTOMER, "CustomerGroup")

// UpdateCustomerGroupsStepInput mirrors UpdateCustomerGroupStepInput.
type UpdateCustomerGroupsStepInput struct {
	Selector *dal.Record `json:"selector"`
	Update   *dal.Record `json:"update"`
}

// UpdateCustomerGroupsStep mirrors updateCustomerGroupsStep.
var UpdateCustomerGroupsStep = composer.CreateStep("update-customer-groups",
	func(ctx composer.StepExecutionContext, in UpdateCustomerGroupsStepInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := customers(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "CustomerGroups", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		u := dal.NewRecord()
		u.Set("selector", in.Selector)
		u.Set("data", in.Update)
		res, err := svc.Internal("CustomerGroup").Update(ctx.Context, u)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		var out []*dal.Record
		if r, ok := res.(*dal.Record); ok {
			out = []*dal.Record{r}
		} else {
			out, _ = res.([]*dal.Record)
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		svc := customers(ctx.Container)
		for _, c := range prev {
			n := dal.NewRecord()
			for _, k := range []string{"id", "name", "metadata"} {
				v, _ := c.Get(k)
				n.Set(k, v)
			}
			if _, err := commonsteps.UpdateRecords(ctx.Context, svc, "CustomerGroup", []any{n}); err != nil {
				return err
			}
		}
		return nil
	})

// GroupCustomerPair is { customer_id, customer_group_id }.
type GroupCustomerPair struct {
	CustomerID      string
	CustomerGroupID string
}

func (p GroupCustomerPair) record() *dal.Record {
	r := dal.NewRecord()
	r.Set("customer_id", p.CustomerID)
	r.Set("customer_group_id", p.CustomerGroupID)
	return r
}

// AddCustomerToGroup mirrors addCustomerToGroup.
func AddCustomerToGroup(ctx context.Context, svc commonsteps.ModuleService, pairs []GroupCustomerPair) error {
	if len(pairs) == 0 {
		return nil
	}
	data := make([]*dal.Record, len(pairs))
	for i, p := range pairs {
		data[i] = p.record()
	}
	_, err := svc.Internal("CustomerGroupCustomer").Create(ctx, data)
	return err
}

// RemoveCustomerFromGroup mirrors removeCustomerFromGroup.
func RemoveCustomerFromGroup(ctx context.Context, svc commonsteps.ModuleService, pairs []GroupCustomerPair) error {
	if len(pairs) == 0 {
		return nil
	}
	or := make([]any, len(pairs))
	for i, p := range pairs {
		or[i] = p.record()
	}
	f := dal.NewRecord()
	f.Set("$or", or)
	rows, err := svc.Internal("CustomerGroupCustomer").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	ids := commonsteps.RecordIDs(rows)
	if len(ids) == 0 {
		return nil
	}
	_, err = svc.Internal("CustomerGroupCustomer").Delete(ctx, ids)
	return err
}

// LinkInput mirrors LinkWorkflowInput.
type LinkInput struct {
	ID     string   `json:"id"`
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

type linkCompensation struct {
	toAdd, toRemove []GroupCustomerPair
}

func linkStep(name string, pair func(id, other string) GroupCustomerPair) *composer.Step[LinkInput, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in LinkInput) (composer.StepResponse[any], error) {
			svc := customers(ctx.Container)
			var toAdd, toRemove []GroupCustomerPair
			for _, o := range in.Add {
				toAdd = append(toAdd, pair(in.ID, o))
			}
			for _, o := range in.Remove {
				toRemove = append(toRemove, pair(in.ID, o))
			}
			if err := AddCustomerToGroup(ctx.Context, svc, toAdd); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := RemoveCustomerFromGroup(ctx.Context, svc, toRemove); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil, any(linkCompensation{toAdd: toAdd, toRemove: toRemove})), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			c, ok := data.(linkCompensation)
			if !ok {
				return nil
			}
			svc := customers(ctx.Container)
			if err := RemoveCustomerFromGroup(ctx.Context, svc, c.toAdd); err != nil {
				return err
			}
			if err := AddCustomerToGroup(ctx.Context, svc, c.toRemove); err != nil {
				return fmt.Errorf("%w", err)
			}
			return nil
		})
}

// LinkCustomersToCustomerGroupStep mirrors linkCustomersToCustomerGroupStep
// (id is the group).
var LinkCustomersToCustomerGroupStep = linkStep("link-customers-to-customer-group", func(id, other string) GroupCustomerPair {
	return GroupCustomerPair{CustomerID: other, CustomerGroupID: id}
})

// LinkCustomerGroupsToCustomerStep mirrors linkCustomerGroupsToCustomerStep
// (id is the customer; upstream reuses the same step id).
var LinkCustomerGroupsToCustomerStep = linkStep("link-customers-to-customer-group", func(id, other string) GroupCustomerPair {
	return GroupCustomerPair{CustomerID: id, CustomerGroupID: other}
})
