// Package steps mirrors @medusajs/core-flows/src/customer/steps.
package steps

import (
	"context"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func customers(c composer.Container) commonsteps.ModuleService {
	return commonsteps.ModuleOf(c, utilsmodulessdk.CUSTOMER)
}

func get(r *dal.Record, k string) any {
	v, _ := r.Get(k)
	return v
}

// CreateCustomers mirrors CustomerModuleService.createCustomers: addresses
// are created after the customers.
func CreateCustomers(ctx context.Context, svc commonsteps.ModuleService, data []*dal.Record) ([]*dal.Record, error) {
	attrs := make([]*dal.Record, len(data))
	for i, d := range data {
		n := dal.NewRecord()
		for _, k := range d.Keys() {
			if k == "addresses" {
				continue
			}
			n.Set(k, get(d, k))
		}
		attrs[i] = n
	}
	created, err := svc.Internal("Customer").Create(ctx, attrs)
	if err != nil {
		return nil, err
	}
	var addresses []*dal.Record
	for i, d := range data {
		list, _ := get(d, "addresses").([]any)
		for _, a := range list {
			ar, ok := a.(*dal.Record)
			if !ok {
				continue
			}
			n := dal.NewRecord()
			for _, k := range ar.Keys() {
				n.Set(k, get(ar, k))
			}
			n.Set("customer_id", get(created[i], "id"))
			addresses = append(addresses, n)
		}
	}
	if len(addresses) > 0 {
		if _, err := svc.Internal("CustomerAddress").Create(ctx, addresses); err != nil {
			return nil, err
		}
	}
	return created, nil
}

// CreateCustomersStep mirrors createCustomersStep.
var CreateCustomersStep = composer.CreateStep("create-customers",
	func(ctx composer.StepExecutionContext, data []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		var created []*dal.Record
		svc := customers(ctx.Container)
		err := inTx(ctx.Context, svc, func(s commonsteps.ModuleService) error {
			var err error
			created, err = CreateCustomers(ctx.Context, s, data)
			return err
		})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := customers(ctx.Container).Internal("Customer").Delete(ctx.Context, ids)
		return err
	})

type transactional interface {
	InTransaction(ctx context.Context, fn func(tx *utilsmodulessdk.MedusaService) error) error
}

func inTx(ctx context.Context, svc commonsteps.ModuleService, fn func(s commonsteps.ModuleService) error) error {
	if t, ok := svc.(transactional); ok {
		return t.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error { return fn(tx) })
	}
	return fn(svc)
}

// SelectorUpdateInput is { selector, update }.
type SelectorUpdateInput struct {
	Selector *dal.Record
	Update   *dal.Record
}

func updateBySelector(ctx context.Context, svc commonsteps.ModuleService, className string, selector, data *dal.Record) ([]*dal.Record, error) {
	u := dal.NewRecord()
	u.Set("selector", selector)
	u.Set("data", data)
	res, err := svc.Internal(className).Update(ctx, u)
	if err != nil {
		return nil, err
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	out, _ := res.([]*dal.Record)
	return out, nil
}

// UpdateCustomersStep mirrors updateCustomersStep.
var UpdateCustomersStep = composer.CreateStep("update-customer",
	func(ctx composer.StepExecutionContext, in SelectorUpdateInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := customers(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "Customers", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := updateBySelector(ctx.Context, svc, "Customer", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		svc := customers(ctx.Container)
		for _, c := range prev {
			n := dal.NewRecord()
			n.Set("id", get(c, "id"))
			for _, k := range []string{"first_name", "last_name", "email", "phone", "metadata"} {
				n.Set(k, get(c, k))
			}
			if _, err := commonsteps.UpdateRecords(ctx.Context, svc, "Customer", []any{n}); err != nil {
				return err
			}
		}
		return nil
	})

// DeleteCustomersStep mirrors deleteCustomersStep.
var DeleteCustomersStep = commonsteps.SoftDeleteRecordsStep("delete-customers", utilsmodulessdk.CUSTOMER, "Customer")

// CreateCustomerAddressesStep mirrors createCustomerAddressesStep.
var CreateCustomerAddressesStep = commonsteps.CreateRecordsStep("create-customer-addresses", utilsmodulessdk.CUSTOMER, "CustomerAddress")

// UpdateCustomerAddressesStep mirrors updateCustomerAddressesStep.
var UpdateCustomerAddressesStep = composer.CreateStep("update-customer-addresses",
	func(ctx composer.StepExecutionContext, in SelectorUpdateInput) (composer.StepResponse[[]*dal.Record], error) {
		svc := customers(ctx.Container)
		prev, err := commonsteps.PrevDataFor(ctx.Context, svc, "CustomerAddresses", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		out, err := updateBySelector(ctx.Context, svc, "CustomerAddress", in.Selector, in.Update)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(out, any(prev)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		svc := customers(ctx.Container)
		for _, c := range prev {
			if _, err := commonsteps.UpdateRecords(ctx.Context, svc, "CustomerAddress", []any{c}); err != nil {
				return err
			}
		}
		return nil
	})

// DeleteCustomerAddressesStep mirrors deleteCustomerAddressesStep (hard
// delete; compensation recreates).
var DeleteCustomerAddressesStep = composer.CreateStep("delete-customer-addresses",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[any], error) {
		svc := customers(ctx.Container)
		f := dal.NewRecord()
		f.Set("id", ids)
		existing, err := svc.ListBySuffix(ctx.Context, "CustomerAddresses", f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if _, err := svc.Internal("CustomerAddress").Delete(ctx.Context, ids); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil, any(existing)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		prev, _ := data.([]*dal.Record)
		if len(prev) == 0 {
			return nil
		}
		_, err := customers(ctx.Container).Internal("CustomerAddress").Create(ctx.Context, prev)
		return err
	})

// MaybeUnsetInput mirrors MaybeUnsetDefault*AddressStepInput.
type MaybeUnsetInput struct {
	Create []*dal.Record
	Update *SelectorUpdateInput
}

func unsetDefault(name, field string) *composer.Step[MaybeUnsetInput, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in MaybeUnsetInput) (composer.StepResponse[any], error) {
			svc := customers(ctx.Container)
			var customerIDs []any
			switch {
			case in.Create != nil:
				for _, a := range in.Create {
					if b, _ := get(a, field).(bool); b {
						customerIDs = append(customerIDs, get(a, "customer_id"))
					}
				}
			case in.Update != nil:
				if b, _ := get(in.Update.Update, field).(bool); !b {
					return composer.NewStepResponse[any](nil), nil
				}
				affected, err := svc.Internal("CustomerAddress").List(ctx.Context, in.Update.Selector, utilsmodulessdk.FindConfig{Select: []string{"id", "customer_id"}})
				if err != nil {
					return composer.StepResponse[any]{}, err
				}
				for _, a := range affected {
					customerIDs = append(customerIDs, get(a, "customer_id"))
				}
			default:
				return composer.StepResponse[any]{}, nil
			}
			if customerIDs == nil {
				customerIDs = []any{}
			}
			f := dal.NewRecord()
			f.Set("customer_id", customerIDs)
			f.Set(field, true)
			defaults, err := svc.Internal("CustomerAddress").List(ctx.Context, f, utilsmodulessdk.FindConfig{})
			if err != nil {
				return composer.StepResponse[any]{}, err
			}
			if len(customerIDs) > 0 {
				d := dal.NewRecord()
				d.Set(field, false)
				if _, err := updateBySelector(ctx.Context, svc, "CustomerAddress", f, d); err != nil {
					return composer.StepResponse[any]{}, err
				}
			}
			return composer.NewStepResponse[any](nil, any(commonsteps.RecordIDs(defaults))), nil
		},
		func(ctx composer.StepExecutionContext, data any) error {
			ids, _ := data.([]any)
			if len(ids) == 0 {
				return nil
			}
			f := dal.NewRecord()
			f.Set("id", ids)
			d := dal.NewRecord()
			d.Set(field, true)
			_, err := updateBySelector(ctx.Context, customers(ctx.Container), "CustomerAddress", f, d)
			return err
		})
}

// MaybeUnsetDefaultShippingAddressesStep mirrors maybeUnsetDefaultShippingAddressesStep.
var MaybeUnsetDefaultShippingAddressesStep = unsetDefault("maybe-unset-default-shipping-customer-addresses", "is_default_shipping")

// MaybeUnsetDefaultBillingAddressesStep mirrors maybeUnsetDefaultBillingAddressesStep.
var MaybeUnsetDefaultBillingAddressesStep = unsetDefault("maybe-unset-default-billing-customer-addresses", "is_default_billing")
