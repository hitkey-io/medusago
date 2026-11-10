package workflows

import (
	"fmt"

	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type returnActionValidation struct {
	Order       *dal.Record
	OrderReturn *dal.Record
	OrderChange *dal.Record
	Input       OrderChangeActionInput
}

// returnActionValidationStep mirrors the validation steps of the return
// action workflows: cancellation checks, an active change and an action of
// one of the allowed types (plain Errors upstream).
func returnActionValidationStep(name, missing, wrong string, withOrder bool, allowed ...string) *composer.Step[returnActionValidation, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in returnActionValidation) (composer.StepResponse[any], error) {
			if withOrder {
				if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
					return composer.StepResponse[any]{}, err
				}
			}
			if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			err := editAction(in.OrderChange, in.Input, missing, func(act string) error {
				for _, a := range allowed {
					if a == act {
						return nil
					}
				}
				return fmt.Errorf(wrong, in.Input.ActionID)
			})
			return composer.NewStepResponse[any](nil), err
		}, nil)
}

func returnChange(e *composer.Execution, orderReturn *dal.Record, fields []string) (*dal.Record, error) {
	return activeOrderChangeFor(e, g(orderReturn, "order_id"), "return_id", g(orderReturn, "id"), fields)
}

func previewFor(e *composer.Execution, orderID any) (*dal.Record, error) {
	return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(orderID))
}

var updateReturnValidationStep = composer.CreateStep("validate-update-return",
	func(ctx composer.StepExecutionContext, in returnActionValidation) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), ThrowIfOrderChangeIsNotActive(in.OrderChange)
	}, nil)

// UpdateReturnWorkflow mirrors updateReturnWorkflow.
var UpdateReturnWorkflow = composer.CreateWorkflow("update-return",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, g(in, "return_id"), []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := updateReturnValidationStep.Run(e, returnActionValidation{OrderReturn: orderReturn, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", g(in, "return_id"))
		setIfPresent(u, in, "location_id", "no_notification", "metadata")
		if _, err := ordersteps.UpdateReturnsStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		return previewFor(e, g(orderReturn, "order_id"))
	})

var cancelReturnValidateOrder = composer.CreateStep("validate-return",
	func(ctx composer.StepExecutionContext, orderReturn *dal.Record) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(orderReturn, "Return"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		for _, f := range recs(g(orderReturn, "fulfillments")) {
			if !truthy(g(f, "canceled_at")) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "All fulfillments must be canceled before canceling a return")
			}
		}
		for _, item := range recs(g(orderReturn, "items")) {
			if totals.Gt(g(item, "received_quantity"), 0) {
				return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Can't cancel a return which has returned items")
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CancelReturnWorkflow mirrors cancelReturnWorkflow (returns nothing).
var CancelReturnWorkflow = composer.CreateWorkflow("cancel-return",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		orderReturn, err := queryReturn(e, g(in, "return_id"), []string{"id", "order_id", "canceled_at", "items.id", "items.received_quantity", "fulfillments.canceled_at"})
		if err != nil {
			return nil, err
		}
		if _, err := cancelReturnValidateOrder.Run(e, orderReturn); err != nil {
			return nil, err
		}
		d := dal.NewRecord()
		d.Set("return_id", g(orderReturn, "id"))
		d.Set("order_id", g(orderReturn, "order_id"))
		if v, ok := in.Get("canceled_by"); ok {
			d.Set("canceled_by", v)
		}
		_, err = ordersteps.CancelOrderReturnStep.Run(e, d)
		return nil, err
	})

var cancelReceiveReturnValidationStep = composer.CreateStep("validate-cancel-return-shipping-method",
	func(ctx composer.StepExecutionContext, in returnActionValidation) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), ThrowIfOrderChangeIsNotActive(in.OrderChange)
	}, nil)

// CancelReturnReceiveWorkflow mirrors cancelReturnReceiveWorkflow.
var CancelReturnReceiveWorkflow = composer.CreateWorkflow("cancel-receive-return",
	func(e *composer.Execution, returnID any) (any, error) {
		orderReturn, err := queryReturn(e, returnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "version", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "version"})
		if err != nil {
			return nil, err
		}
		if _, err := cancelReceiveReturnValidationStep.Run(e, returnActionValidation{Order: order, OrderReturn: orderReturn, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		_, err = ordersteps.DeleteOrderChangesStep.Run(e, []any{g(orderChange, "id")})
		return nil, err
	})

// ReturnActionInput mirrors the { return_id, action_id, data } inputs.
type ReturnActionInput struct {
	ReturnID any
	ActionID any
	Data     *dal.Record
	// ClaimID narrows the order change of updateRequestItemReturnWorkflow
	// (input.claim_id; the exchange routes pass exchange_id, which it ignores).
	ClaimID any
}

func (in ReturnActionInput) action() OrderChangeActionInput {
	return OrderChangeActionInput{OrderID: in.ReturnID, ActionID: in.ActionID, Data: in.Data}
}

var updateRequestItemReturnValidationStep = composer.CreateStep("update-request-item-return-validation",
	func(ctx composer.StepExecutionContext, in returnActionValidation) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		err := editAction(in.OrderChange, in.Input, "No request return found for return %v in order change %v", func(act string) error {
			if act != "RETURN_ITEM" {
				return fmt.Errorf("Action %v is not requesting item return", in.Input.ActionID)
			}
			return nil
		})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		if reason := g(in.Input.Data, "reason_id"); truthy(reason) {
			item := dal.NewRecord()
			item.Set("reason_id", reason)
			if err := validateReturnReasons(ctx, g(in.Order, "id"), []*dal.Record{item}); err != nil {
				return composer.StepResponse[any]{}, err
			}
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// UpdateRequestItemReturnWorkflow mirrors updateRequestItemReturnWorkflow
// (the order change is looked up by order and the undefined claim_id).
var UpdateRequestItemReturnWorkflow = composer.CreateWorkflow("update-request-item-return",
	func(e *composer.Execution, in ReturnActionInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		f := dal.NewRecord()
		f.Set("order_id", g(orderReturn, "order_id"))
		if in.ClaimID != nil {
			f.Set("claim_id", in.ClaimID)
		}
		f.Set("status", []any{"pending", "requested"})
		v := dal.NewRecord()
		v.Set("filters", f)
		orderChange, err := remoteOne(e, "order-change-query", "order_change", v, []string{"id", "status", "version", "actions.*"}, false)
		if err != nil {
			return nil, err
		}
		if _, err := updateRequestItemReturnValidationStep.Run(e, returnActionValidation{order, orderReturn, orderChange, in.action()}); err != nil {
			return nil, err
		}
		original := rec(g(FindChangeAction(orderChange, in.ActionID), "details"))
		u := dal.NewRecord()
		u.Set("id", in.ActionID)
		d := dal.NewRecord()
		if q := g(in.Data, "quantity"); q != nil {
			d.Set("quantity", q)
		} else if q := g(original, "quantity"); q != nil {
			d.Set("quantity", q)
		}
		if r, ok := in.Data.Get("reason_id"); ok && r != nil {
			d.Set("reason_id", r)
		} else if r, ok := original.Get("reason_id"); ok {
			d.Set("reason_id", r)
		}
		u.Set("details", d)
		setIfPresent(u, in.Data, "internal_note")
		if _, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		args := dal.NewRecord()
		args.Set("order_change_id", g(orderChange, "id"))
		args.Set("return_id", g(orderReturn, "id"))
		args.Set("order_id", g(orderReturn, "order_id"))
		if _, err := RefreshReturnShippingWorkflow.RunAsStep(e, args); err != nil {
			return nil, err
		}
		return previewFor(e, g(order, "id"))
	})

var removeReturnItemActionValidationStep = returnActionValidationStep("remove-item-return-action-validation",
	"No request return found for return %v in order change %v", "Action %v is not requesting item return", true, "RETURN_ITEM")

// RemoveItemReturnActionWorkflow mirrors removeItemReturnActionWorkflow.
var RemoveItemReturnActionWorkflow = composer.CreateWorkflow("remove-item-return-action",
	func(e *composer.Execution, in ReturnActionInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "version", "return_id", "order_id", "actions.*", "canceled_at", "confirmed_at", "declined_at"})
		if err != nil {
			return nil, err
		}
		if _, err := removeReturnItemActionValidationStep.Run(e, returnActionValidation{order, orderReturn, orderChange, in.action()}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{in.ActionID}); err != nil {
			return nil, err
		}
		f := dal.NewRecord()
		f.Set("order_id", g(orderReturn, "order_id"))
		f.Set("return_id", g(orderReturn, "id"))
		f.Set("status", []any{"pending", "requested"})
		v := dal.NewRecord()
		v.Set("filters", f)
		updated, err := remoteOne(e, "updated-order-change-query", "order_change", v, []string{"actions.id", "actions.action", "actions.order_id",
			"actions.return_id", "actions.claim_id", "actions.exchange_id", "actions.reference", "actions.reference_id", "actions.order_change_id"}, false)
		if err != nil {
			return nil, err
		}
		var toDelete any
		hasItems := false
		for _, a := range recs(g(updated, "actions")) {
			if g(a, "action") == "RETURN_ITEM" {
				hasItems = true
			}
		}
		if !hasItems {
			for _, a := range recs(g(updated, "actions")) {
				if g(a, "action") == "SHIPPING_ADD" && g(a, "return_id") == g(orderReturn, "id") {
					toDelete = g(a, "id")
					break
				}
			}
		}
		if truthy(toDelete) {
			if _, err := RemoveReturnShippingMethodWorkflow.RunAsStep(e, ReturnActionInput{ReturnID: g(orderReturn, "id"), ActionID: toDelete}); err != nil {
				return nil, err
			}
			u := dal.NewRecord()
			u.Set("return_id", g(orderReturn, "id"))
			u.Set("location_id", nil)
			if _, err := UpdateReturnWorkflow.RunAsStep(e, u); err != nil {
				return nil, err
			}
		} else {
			args := dal.NewRecord()
			args.Set("order_change_id", g(orderChange, "id"))
			args.Set("return_id", g(orderReturn, "id"))
			args.Set("order_id", g(orderReturn, "order_id"))
			if _, err := RefreshReturnShippingWorkflow.RunAsStep(e, args); err != nil {
				return nil, err
			}
		}
		return previewFor(e, g(order, "id"))
	})

var updateReceiveItemReturnRequestValidationStep = returnActionValidationStep("update-receive-item-return-request-validation",
	"No request return found for return %v in order change %v", "Action %v is not receiving an item return request", true,
	"RECEIVE_RETURN_ITEM", "RECEIVE_DAMAGED_RETURN_ITEM")

// UpdateReceiveItemReturnRequestWorkflow mirrors updateReceiveItemReturnRequestWorkflow.
var UpdateReceiveItemReturnRequestWorkflow = composer.CreateWorkflow("update-receive-item-return-request",
	func(e *composer.Execution, in ReturnActionInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := updateReceiveItemReturnRequestValidationStep.Run(e, returnActionValidation{order, orderReturn, orderChange, in.action()}); err != nil {
			return nil, err
		}
		original := rec(g(FindChangeAction(orderChange, in.ActionID), "details"))
		u := dal.NewRecord()
		u.Set("id", in.ActionID)
		d := dal.NewRecord()
		if q := g(in.Data, "quantity"); q != nil {
			d.Set("quantity", q)
		} else if q := g(original, "quantity"); q != nil {
			d.Set("quantity", q)
		}
		u.Set("details", d)
		setIfPresent(u, in.Data, "internal_note")
		if _, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		return previewFor(e, g(order, "id"))
	})

var removeItemReceiveReturnActionValidationStep = returnActionValidationStep("remove-item-receive-return-action-validation",
	"No request return found for return %v in order change %v", "Action %v is not receiving item return", true,
	"RECEIVE_RETURN_ITEM", "RECEIVE_DAMAGED_RETURN_ITEM")

// RemoveItemReceiveReturnActionWorkflow mirrors removeItemReceiveReturnActionWorkflow.
var RemoveItemReceiveReturnActionWorkflow = composer.CreateWorkflow("remove-item-receive-return-action",
	func(e *composer.Execution, in ReturnActionInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := removeItemReceiveReturnActionValidationStep.Run(e, returnActionValidation{order, orderReturn, orderChange, in.action()}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.DeleteOrderChangeActionsStep.Run(e, []any{in.ActionID}); err != nil {
			return nil, err
		}
		return previewFor(e, g(order, "id"))
	})

var dismissItemReturnRequestValidationStep = composer.CreateStep("dismiss-item-return-request-validation",
	func(ctx composer.StepExecutionContext, in requestValidationInput) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), throwIfItemsDoesNotExistsInReturn(in.OrderReturn, in.Items)
	}, nil)

// DismissItemReturnRequestWorkflow mirrors dismissItemReturnRequestWorkflow.
var DismissItemReturnRequestWorkflow = composer.CreateWorkflow("dismiss-item-return-request",
	func(e *composer.Execution, in RequestItemReturnInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "order_id", "return_id"})
		if err != nil {
			return nil, err
		}
		if _, err := dismissItemReturnRequestValidationStep.Run(e, requestValidationInput{order, orderReturn, orderChange, in.Items}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("return_id", g(orderReturn, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "RECEIVE_DAMAGED_RETURN_ITEM")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "return")
			a.Set("reference_id", g(orderReturn, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			d.Set("quantity", g(item, "quantity"))
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		return previewFor(e, g(order, "id"))
	})

var updateReturnShippingMethodValidationStep = returnActionValidationStep("validate-update-return-shipping-method",
	"No shipping method found for return %v in order change %v", "Action %v is not adding a shipping method", false, "SHIPPING_ADD")

// UpdateReturnShippingMethodWorkflow mirrors updateReturnShippingMethodWorkflow.
var UpdateReturnShippingMethodWorkflow = composer.CreateWorkflow("update-return-shipping-method",
	func(e *composer.Execution, in ReturnActionInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at", "order.currency_code"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order_return", orderReturn)
		hookIn.Set("order_change", orderChange)
		hookIn.Set("additional_data", g(in.Data, "additional_data"))
		if err := composer.CreateHook(e, "setPricingContext", hookIn); err != nil {
			return nil, err
		}
		var options []*dal.Record
		if customAmountNull(in.Data) {
			original := FindChangeAction(orderChange, in.ActionID)
			if options, err = FetchShippingOptionsForMethod(e, g(original, "reference_id"), g(rec(g(orderReturn, "order")), "currency_code")); err != nil {
				return nil, err
			}
		}
		if _, err := updateReturnShippingMethodValidationStep.Run(e, returnActionValidation{OrderReturn: orderReturn, OrderChange: orderChange, Input: in.action()}); err != nil {
			return nil, err
		}
		if err := UpdateShippingMethodAction(e, in.action(), orderChange, options); err != nil {
			return nil, err
		}
		return previewFor(e, g(orderReturn, "order_id"))
	})

var removeReturnShippingMethodValidationStep = returnActionValidationStep("validate-remove-return-shipping-method",
	"No shipping method found for return %v in order change %v", "Action %v is not adding a shipping method", false, "SHIPPING_ADD")

// RemoveReturnShippingMethodWorkflow mirrors removeReturnShippingMethodWorkflow.
var RemoveReturnShippingMethodWorkflow = composer.CreateWorkflow("remove-return-shipping-method",
	func(e *composer.Execution, in ReturnActionInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := returnChange(e, orderReturn, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := removeReturnShippingMethodValidationStep.Run(e, returnActionValidation{OrderReturn: orderReturn, OrderChange: orderChange, Input: in.action()}); err != nil {
			return nil, err
		}
		if err := RemoveShippingMethodAction(e, orderChange, in.ActionID); err != nil {
			return nil, err
		}
		return previewFor(e, g(orderReturn, "order_id"))
	})
