package workflows

import (
	"errors"
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// OrderChangeActionInput mirrors the { order_id, action_id, data } inputs
// of the order change action workflows (return, claim and exchange ids go
// in OrderID as the entity id).
type OrderChangeActionInput struct {
	OrderID  any
	ActionID any
	Data     *dal.Record
}

// FindChangeAction returns the action of an order change by id.
func FindChangeAction(orderChange *dal.Record, id any) *dal.Record {
	for _, a := range recs(g(orderChange, "actions")) {
		if g(a, "id") == id {
			return a
		}
	}
	return nil
}

// ShippingMethodUpdate is the result of PrepareShippingMethodUpdate.
type ShippingMethodUpdate struct {
	Action         *dal.Record
	ShippingMethod *dal.Record
}

// PrepareShippingMethodUpdate mirrors prepareShippingMethodUpdate: options
// is nil when the custom amount is used (shippingOptions undefined).
func PrepareShippingMethodUpdate(in OrderChangeActionInput, orderChange *dal.Record, options []*dal.Record) ShippingMethodUpdate {
	original := FindChangeAction(orderChange, in.ActionID)
	isCustom := options == nil
	var price any
	if isCustom {
		price = g(in.Data, "custom_amount")
	} else if len(options) > 0 {
		price = g(rec(g(options[0], "calculated_price")), "calculated_amount")
	}
	action := dal.NewRecord()
	action.Set("id", g(original, "id"))
	if price != nil || !isCustom || in.Data.Has("custom_amount") {
		action.Set("amount", price)
	}
	setIfPresent(action, in.Data, "internal_note")
	method := dal.NewRecord()
	method.Set("id", g(original, "reference_id"))
	if price != nil || !isCustom || in.Data.Has("custom_amount") {
		method.Set("amount", price)
	}
	method.Set("is_custom_amount", isCustom)
	setIfPresent(method, in.Data, "metadata")
	return ShippingMethodUpdate{Action: action, ShippingMethod: method}
}

func editAction(orderChange *dal.Record, in OrderChangeActionInput, missing string, check func(action string) error) error {
	if err := ThrowIfOrderChangeIsNotActive(orderChange); err != nil {
		return err
	}
	a := FindChangeAction(orderChange, in.ActionID)
	if a == nil {
		return errors.New(fmt.Sprintf(missing, in.OrderID, g(orderChange, "id")))
	}
	act, _ := g(a, "action").(string)
	return check(act)
}

type editActionValidation struct {
	Order       *dal.Record
	OrderChange *dal.Record
	Input       OrderChangeActionInput
}

var updateOrderEditAddItemValidationStep = composer.CreateStep("update-order-edit-add-item-validation",
	func(ctx composer.StepExecutionContext, in editActionValidation) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		err := editAction(in.OrderChange, in.Input, "No request to add item for order %v in order change %v", func(act string) error {
			if act != "ITEM_ADD" {
				return fmt.Errorf("Action %v is not adding an item", in.Input.ActionID)
			}
			return nil
		})
		return composer.NewStepResponse[any](nil), err
	}, nil)

// UpdateOrderEditAddItemWorkflow mirrors updateOrderEditAddItemWorkflow.
var UpdateOrderEditAddItemWorkflow = composer.CreateWorkflow("update-order-edit-add-item", withOrderLock(func(in OrderChangeActionInput) any { return in.OrderID },
	func(e *composer.Execution, in OrderChangeActionInput) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "order-query", in.OrderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		if _, err := updateOrderEditAddItemValidationStep.Run(e, editActionValidation{order, orderChange, in}); err != nil {
			return nil, err
		}
		original := rec(g(FindChangeAction(orderChange, in.ActionID), "details"))
		u := dal.NewRecord()
		u.Set("id", in.ActionID)
		d := dal.NewRecord()
		for _, k := range []string{"quantity", "unit_price", "compare_at_unit_price"} {
			v := g(in.Data, k)
			if v == nil {
				v = g(original, k)
			}
			if v != nil {
				d.Set(k, v)
			}
		}
		u.Set("details", d)
		setIfPresent(u, in.Data, "internal_note")
		if _, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
	}))

var removeOrderEditItemActionValidationStep = composer.CreateStep("remove-item-order-edit-action-validation",
	func(ctx composer.StepExecutionContext, in editActionValidation) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		err := editAction(in.OrderChange, in.Input, "No item found for order %v in order change %v", func(act string) error {
			if act != "ITEM_ADD" && act != "ITEM_UPDATE" {
				return fmt.Errorf("Action %v is not adding or updating an item", in.Input.ActionID)
			}
			return nil
		})
		return composer.NewStepResponse[any](nil), err
	}, nil)

// RemoveItemOrderEditActionWorkflow mirrors removeItemOrderEditActionWorkflow.
var RemoveItemOrderEditActionWorkflow = composer.CreateWorkflow("remove-item-order edit-action", withOrderLock(func(in OrderChangeActionInput) any { return in.OrderID },
	func(e *composer.Execution, in OrderChangeActionInput) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "order-query", in.OrderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		if _, err := removeOrderEditItemActionValidationStep.Run(e, editActionValidation{order, orderChange, in}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{in.ActionID}); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
	}))

func shippingActionValidationStep(name string) *composer.Step[editActionValidation, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in editActionValidation) (composer.StepResponse[any], error) {
			err := editAction(in.OrderChange, in.Input, "No shipping method found for order %v in order change %v", func(act string) error {
				if act != "SHIPPING_ADD" {
					return fmt.Errorf("Action %v is not adding a shipping method", in.Input.ActionID)
				}
				return nil
			})
			return composer.NewStepResponse[any](nil), err
		}, nil)
}

var updateOrderEditShippingMethodValidationStep = shippingActionValidationStep("validate-update-order-edit-shipping-method")

// FetchShippingOptionsForMethod mirrors the "fetch-shipping-method" and
// "fetch-shipping-option" remote queries of the shipping method action
// update workflows (flat prices in the currency of the order).
func FetchShippingOptionsForMethod(e *composer.Execution, methodID, currencyCode any) ([]*dal.Record, error) {
	mv := dal.NewRecord()
	mv.Set("id", methodID)
	method, err := remoteOne(e, "fetch-shipping-method", "order_shipping_method", mv, []string{"id", "shipping_option_id"}, false)
	if err != nil {
		return nil, err
	}
	pc := dal.NewRecord()
	pc.Set("currency_code", currencyCode)
	cp := dal.NewRecord()
	cp.Set("context", pc)
	v := dal.NewRecord()
	v.Set("id", g(method, "shipping_option_id"))
	v.Set("calculated_price", cp)
	res, err := commonsteps.UseRemoteQueryStep.Config("fetch-shipping-option", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
		EntryPoint: "shipping_option", Variables: v,
		Fields: []string{"id", "name", "calculated_price.calculated_amount", "calculated_price.is_calculated_price_tax_inclusive"},
	})
	if err != nil {
		return nil, err
	}
	out := recs(res.Data)
	if out == nil {
		out = []*dal.Record{}
	}
	return out, nil
}

// UpdateShippingMethodAction applies PrepareShippingMethodUpdate (the
// action and the shipping method are updated in parallel).
func UpdateShippingMethodAction(e *composer.Execution, in OrderChangeActionInput, orderChange *dal.Record, options []*dal.Record) error {
	upd := PrepareShippingMethodUpdate(in, orderChange, options)
	return composer.Parallelize(
		func() error {
			_, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{upd.Action})
			return err
		},
		func() error {
			_, err := ordersteps.UpdateOrderShippingMethodsStep.Run(e, []*dal.Record{upd.ShippingMethod})
			return err
		},
	)
}

// RemoveShippingMethodAction deletes a SHIPPING_ADD action and its method.
func RemoveShippingMethodAction(e *composer.Execution, orderChange *dal.Record, actionID any) error {
	a := FindChangeAction(orderChange, actionID)
	return composer.Parallelize(
		func() error {
			_, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{g(a, "id")})
			return err
		},
		func() error {
			_, err := ordersteps.DeleteOrderShippingMethodsStep.Run(e, []any{g(a, "reference_id")})
			return err
		},
	)
}

func customAmountNull(data *dal.Record) bool {
	v, ok := data.Get("custom_amount")
	return ok && v == nil
}

// UpdateOrderEditShippingMethodWorkflow mirrors updateOrderEditShippingMethodWorkflow.
var UpdateOrderEditShippingMethodWorkflow = composer.CreateWorkflow("update-order-edit-shipping-method", withOrderLock(func(in OrderChangeActionInput) any { return in.OrderID },
	func(e *composer.Execution, in OrderChangeActionInput) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "order-query", in.OrderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order", order)
		hookIn.Set("order_change", orderChange)
		hookIn.Set("additional_data", g(in.Data, "additional_data"))
		if err := composer.CreateHook(e, "setPricingContext", hookIn); err != nil {
			return nil, err
		}
		var options []*dal.Record
		if customAmountNull(in.Data) {
			original := FindChangeAction(orderChange, in.ActionID)
			if options, err = FetchShippingOptionsForMethod(e, g(original, "reference_id"), g(order, "currency_code")); err != nil {
				return nil, err
			}
		}
		if _, err := updateOrderEditShippingMethodValidationStep.Run(e, editActionValidation{order, orderChange, in}); err != nil {
			return nil, err
		}
		if err := UpdateShippingMethodAction(e, in, orderChange, options); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(in.OrderID))
	}))

var removeOrderEditShippingMethodValidationStep = shippingActionValidationStep("validate-remove-order-edit-shipping-method")

// RemoveOrderEditShippingMethodWorkflow mirrors removeOrderEditShippingMethodWorkflow.
var RemoveOrderEditShippingMethodWorkflow = composer.CreateWorkflow("remove-order-edit-shipping-method", withOrderLock(func(in OrderChangeActionInput) any { return in.OrderID },
	func(e *composer.Execution, in OrderChangeActionInput) (*dal.Record, error) {
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := removeOrderEditShippingMethodValidationStep.Run(e, editActionValidation{nil, orderChange, in}); err != nil {
			return nil, err
		}
		if err := RemoveShippingMethodAction(e, orderChange, in.ActionID); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(in.OrderID))
	}))
