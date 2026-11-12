package workflows

import (
	"fmt"
	"time"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	reservationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/reservation/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// FieldsToRefreshOrderEdit mirrors fieldsToRefreshOrderEdit.
var FieldsToRefreshOrderEdit = []string{"id", "status", "version", "currency_code", "email", "canceled_at", "sales_channel_id", "region_id",
	"customer_id", "customer.*", "customer.groups.*", "promotions.*", "subtotal", "items.*", "items.subtotal", "items.product.id",
	"items.product_id", "items.adjustments.*", "shipping_methods.*", "shipping_methods.adjustments.*", "shipping_methods.shipping_option_id",
	"shipping_address.*"}

// RequiredOrderFieldsForInventoryConfirmation mirrors requiredOrderFieldsForInventoryConfirmation.
var RequiredOrderFieldsForInventoryConfirmation = []string{"id", "version", "canceled_at", "sales_channel_id", "items.*",
	"items.variant.manage_inventory", "items.variant.allow_backorder", "items.variant.inventory_items.inventory_item_id",
	"items.variant.inventory_items.required_quantity", "items.variant.inventory_items.inventory.location_levels.stocked_quantity",
	"items.variant.inventory_items.inventory.location_levels.reserved_quantity",
	"items.variant.inventory_items.inventory.location_levels.raw_stocked_quantity",
	"items.variant.inventory_items.inventory.location_levels.raw_reserved_quantity",
	"items.variant.inventory_items.inventory.location_levels.location_id",
	"items.variant.inventory_items.inventory.location_levels.stock_locations.id",
	"items.variant.inventory_items.inventory.location_levels.stock_locations.name",
	"items.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.id",
	"items.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.name"}

// queryOrderForEdit mirrors useQueryGraphStep({ entity: "order", fields:
// fieldsToRefreshOrderEdit, options: { throwIfKeyNotFound } }).data[0].
func queryOrderForEdit(e *composer.Execution, name string, orderID any) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", orderID)
	step := commonsteps.UseQueryGraphStep
	if name != "" {
		step = step.Config(name, composer.StepOptions{})
	}
	res, err := step.Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: "order", Fields: FieldsToRefreshOrderEdit, Filters: f},
		Options:    joiner.Options{ThrowIfKeyNotFound: true},
	})
	if err != nil {
		return nil, err
	}
	if list := recs(res.Data); len(list) > 0 {
		return list[0], nil
	}
	return nil, nil
}

func editActiveChange(e *composer.Execution, orderID any, fields []string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("order_id", orderID)
	f.Set("status", []any{"pending", "requested"})
	list, err := queryList(e, "order-change-query", "order_change", f, fields)
	if err != nil {
		return nil, err
	}
	if len(list) > 0 {
		return list[0], nil
	}
	return nil, nil
}

func editValidationStep(name string) *composer.Step[changeValidationInput, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in changeValidationInput) (composer.StepResponse[any], error) {
			if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil), nil
		}, nil)
}

var beginOrderEditValidationStep = composer.CreateStep("begin-order-edit-validation",
	func(ctx composer.StepExecutionContext, order *dal.Record) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// BeginOrderEditOrderWorkflow mirrors beginOrderEditOrderWorkflow.
var BeginOrderEditOrderWorkflow = composer.CreateWorkflow("begin-order-edit-order", withOrderLock(func(in *dal.Record) any { return g(in, "order_id") },
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "", g(in, "order_id"))
		if err != nil {
			return nil, err
		}
		if _, err := beginOrderEditValidationStep.Run(e, order); err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("change_type", "edit")
		c.Set("order_id", g(in, "order_id"))
		setIfPresent(c, in, "created_by", "description", "internal_note")
		return ordersteps.CreateOrderChangeStep.Run(e, c)
	}))

var orderEditAddNewItemValidationStep = editValidationStep("order-edit-add-new-item-validation")

// OrderEditItemsInput mirrors the order edit item workflow inputs.
type OrderEditItemsInput struct {
	OrderID any
	Items   []*dal.Record
}

// OrderEditAddNewItemWorkflow mirrors orderEditAddNewItemWorkflow.
var OrderEditAddNewItemWorkflow = composer.CreateWorkflow("order-edit-add-new-item", withOrderLock(func(in OrderEditItemsInput) any { return in.OrderID },
	func(e *composer.Execution, in OrderEditItemsInput) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "order-query", in.OrderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		if _, err := orderEditAddNewItemValidationStep.Run(e, changeValidationInput{Order: order, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		lineItems, err := AddOrderLineItemsWorkflow.RunAsStep(e, AddOrderLineItemsInput{OrderID: g(order, "id"), Items: in.Items})
		if err != nil {
			return nil, err
		}
		itemIDs := []any{}
		for _, li := range lineItems {
			itemIDs = append(itemIDs, g(li, "id"))
		}
		if _, err := UpdateOrderTaxLinesWorkflow.RunAsStep(e, UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ItemIDs: itemIDs}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for i, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "ITEM_ADD")
			setIfPresent(a, item, "internal_note")
			d := dal.NewRecord()
			d.Set("reference_id", g(lineItems[i], "id"))
			d.Set("quantity", g(item, "quantity"))
			price := g(item, "unit_price")
			if price == nil {
				price = g(lineItems[i], "unit_price")
			}
			d.Set("unit_price", price)
			cmp := g(item, "compare_at_unit_price")
			if cmp == nil {
				cmp = g(lineItems[i], "compare_at_unit_price")
			}
			d.Set("compare_at_unit_price", cmp)
			setIfPresent(d, item, "allow_backorder", "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(in.OrderID))
	}))

var orderEditUpdateItemQuantityValidationStep = editValidationStep("order-edit-update-item-quantity-validation")

// OrderEditUpdateItemQuantityWorkflow mirrors orderEditUpdateItemQuantityWorkflow.
var OrderEditUpdateItemQuantityWorkflow = composer.CreateWorkflow("order-edit-update-item-quantity", withOrderLock(func(in OrderEditItemsInput) any { return in.OrderID },
	func(e *composer.Execution, in OrderEditItemsInput) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "order-query", in.OrderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		if _, err := orderEditUpdateItemQuantityValidationStep.Run(e, changeValidationInput{Order: order, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			var existing *dal.Record
			for _, oi := range recs(g(order, "items")) {
				if g(oi, "id") == g(item, "id") {
					existing = oi
					break
				}
			}
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "ITEM_UPDATE")
			setIfPresent(a, item, "internal_note")
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			d.Set("quantity", g(item, "quantity"))
			setIfPresent(d, item, "unit_price", "compare_at_unit_price")
			d.Set("quantity_diff", totals.JSONNumber(totals.Sub(g(item, "quantity"), g(existing, "quantity"))))
			setIfPresent(d, item, "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(in.OrderID))
	}))

// CreateOrderEditShippingMethodWorkflow mirrors createOrderEditShippingMethodWorkflow.
var CreateOrderEditShippingMethodWorkflow = composer.CreateWorkflow("create-order-edit-shipping-method", withOrderLock(func(in *dal.Record) any { return g(in, "order_id") },
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		order, err := queryOrderOne(e, g(in, "order_id"), []string{"id", "status", "currency_code", "canceled_at", "locale"})
		if err != nil {
			return nil, err
		}
		hookIn := dal.NewRecord()
		hookIn.Set("order", order)
		hookIn.Set("shipping_option_id", g(in, "shipping_option_id"))
		hookIn.Set("additional_data", g(in, "additional_data"))
		if _, err := composer.CreateHookWithResult(e, "setPricingContext", hookIn); err != nil {
			return nil, err
		}
		pc := dal.NewRecord()
		pc.Set("currency_code", g(order, "currency_code"))
		cp := dal.NewRecord()
		cp.Set("context", pc)
		v := dal.NewRecord()
		v.Set("id", g(in, "shipping_option_id"))
		v.Set("calculated_price", cp)
		res, err := commonsteps.UseRemoteQueryStep.Config("fetch-shipping-option", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
			EntryPoint: "shipping_option", Variables: v,
			Fields: []string{"id", "name", "calculated_price.calculated_amount", "calculated_price.is_calculated_price_tax_inclusive"},
		})
		if err != nil {
			return nil, err
		}
		options := recs(res.Data)
		f := dal.NewRecord()
		f.Set("order_id", g(in, "order_id"))
		f.Set("status", []any{"pending", "requested"})
		vv := dal.NewRecord()
		vv.Set("filters", f)
		orderChange, err := remoteOne(e, "order-change-query", "order_change", vv, []string{"id", "status", "version"}, false)
		if err != nil {
			return nil, err
		}
		var option *dal.Record
		if len(options) > 0 {
			option = options[0]
		}
		if option, err = TranslateShippingOption(e, option, g(order, "locale")); err != nil {
			return nil, err
		}
		related := dal.NewRecord()
		related.Set("order_id", g(order, "id"))
		customPrice := g(in, "custom_amount")
		method := PrepareShippingMethod("", option, orderChange, related, in, customPrice)
		created, err := ordersteps.CreateOrderShippingMethodsStep.Run(e, []*dal.Record{method})
		if err != nil {
			return nil, err
		}
		var methodIDs []any
		for _, m := range created {
			methodIDs = append(methodIDs, g(m, "id"))
		}
		if _, err := UpdateOrderTaxLinesWorkflow.RunAsStep(e, UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ShippingMethodIDs: methodIDs}); err != nil {
			return nil, err
		}
		price := customPrice
		if price == nil {
			price = g(rec(g(option, "calculated_price")), "calculated_amount")
		}
		a := dal.NewRecord()
		a.Set("action", "SHIPPING_ADD")
		a.Set("reference", "order_shipping_method")
		a.Set("order_change_id", g(orderChange, "id"))
		a.Set("reference_id", g(created[0], "id"))
		a.Set("amount", price)
		a.Set("order_id", g(order, "id"))
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{a}); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
	}))

var requestOrderEditRequestValidationStep = editValidationStep("validate-order-edit-request")

// OrderEditRequestInput mirrors OrderEditRequestWorkflowInput.
type OrderEditRequestInput struct {
	OrderID        any
	RequestedBy    any
	NoNotification any
}

// RequestOrderEditRequestWorkflow mirrors requestOrderEditRequestWorkflow.
var RequestOrderEditRequestWorkflow = composer.CreateWorkflow("order-edit-request", withOrderLock(func(in OrderEditRequestInput) any { return in.OrderID },
	func(e *composer.Execution, in OrderEditRequestInput) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "order-query", in.OrderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := requestOrderEditRequestValidationStep.Run(e, changeValidationInput{Order: order, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		u.Set("id", g(orderChange, "id"))
		u.Set("status", "requested")
		u.Set("requested_at", time.Now())
		if in.RequestedBy != nil {
			u.Set("requested_by", in.RequestedBy)
		}
		if in.NoNotification != nil {
			u.Set("no_notification", in.NoNotification)
		}
		if _, err := ordersteps.UpdateOrderChangesStep.Run(e, []*dal.Record{u}); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("actions", g(orderChange, "actions"))
		if in.NoNotification != nil {
			ev.Set("no_notification", in.NoNotification)
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order-edit.requested", Data: ev}); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
	}))

var confirmOrderEditRequestValidationStep = editValidationStep("confirm-order-edit-request-validation")

// ConfirmOrderEditRequestInput mirrors ConfirmOrderEditRequestWorkflowInput.
type ConfirmOrderEditRequestInput struct {
	OrderID     any
	ConfirmedBy any
}

// ConfirmOrderEditRequestWorkflow mirrors confirmOrderEditRequestWorkflow.
var ConfirmOrderEditRequestWorkflow = composer.CreateWorkflow("confirm-order-edit-request", withOrderLock(func(in ConfirmOrderEditRequestInput) any { return in.OrderID },
	func(e *composer.Execution, in ConfirmOrderEditRequestInput) (*dal.Record, error) {
		order, err := queryOrderForEdit(e, "order-query", in.OrderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, in.OrderID, []string{"id", "status", "no_notification", "actions.id", "actions.order_id", "actions.return_id",
			"actions.action", "actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"})
		if err != nil {
			return nil, err
		}
		if _, err := confirmOrderEditRequestValidationStep.Run(e, changeValidationInput{Order: order, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		preview, err := ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
		if err != nil {
			return nil, err
		}
		if _, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}, ConfirmedBy: in.ConfirmedBy}); err != nil {
			return nil, err
		}
		f := dal.NewRecord()
		f.Set("id", in.OrderID)
		isList := false
		res, err := commonsteps.UseQueryGraphStep.Config("order-items-query", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "order", Fields: cartutils.Deduplicate(RequiredOrderFieldsForInventoryConfirmation, FieldsToRefreshOrderEdit), Filters: f},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		refreshed := rec(res.Data)

		backorder := map[any]bool{}
		for _, a := range recs(g(orderChange, "actions")) {
			d := rec(g(a, "details"))
			if g(a, "action") == "ITEM_ADD" && truthy(g(d, "allow_backorder")) {
				backorder[g(d, "reference_id")] = true
			}
		}
		current := map[any]bool{}
		for _, it := range recs(g(refreshed, "items")) {
			current[g(it, "id")] = true
		}
		toRemove := []any{}
		for _, it := range recs(g(order, "items")) {
			if !current[g(it, "id")] {
				toRemove = append(toRemove, g(it, "id"))
			}
		}
		var updated []any
		var items, variants []*dal.Record
		for _, ordItem := range recs(g(refreshed, "items")) {
			var itemAction *dal.Record
			for _, pi := range recs(g(preview, "items")) {
				if g(pi, "id") != g(ordItem, "id") {
					continue
				}
				for _, a := range recs(g(pi, "actions")) {
					if g(a, "action") == "ITEM_ADD" || g(a, "action") == "ITEM_UPDATE" {
						itemAction = pi
						break
					}
				}
				if itemAction != nil {
					break
				}
			}
			if itemAction == nil {
				continue
			}
			for _, a := range recs(g(itemAction, "actions")) {
				if g(a, "action") == "ITEM_UPDATE" {
					updated = append(updated, g(ordItem, "id"))
					break
				}
			}
			newQuantity := g(itemAction, "raw_quantity")
			if newQuantity == nil {
				newQuantity = g(itemAction, "quantity")
			}
			it := dal.NewRecord()
			it.Set("id", g(ordItem, "id"))
			it.Set("variant_id", g(ordItem, "variant_id"))
			it.Set("quantity", totals.Sub(newQuantity, g(ordItem, "raw_fulfilled_quantity")))
			it.Set("allow_backorder", backorder[g(ordItem, "id")])
			items = append(items, it)
			variants = append(variants, rec(g(ordItem, "variant")))
		}
		toRemove = append(toRemove, updated...)
		channelID, _ := g(refreshed, "sales_channel_id").(string)
		inv, err := cartutils.PrepareConfirmInventoryInput(cartutils.ConfirmVariantInventoryInput{SalesChannelID: channelID, Variants: variants, Items: items})
		if err != nil {
			return nil, err
		}
		deletedIDs, err := reservationsteps.DeleteReservationsByLineItemsStep.Run(e, toRemove)
		if err != nil {
			return nil, err
		}
		created, err := cartsteps.ReserveInventoryStep.Run(e, inv)
		if err != nil {
			return nil, err
		}
		deletedEvents, createdEvents := []any{}, []any{}
		for _, id := range deletedIDs {
			ev := dal.NewRecord()
			ev.Set("id", id)
			ev.Set("order_id", g(order, "id"))
			deletedEvents = append(deletedEvents, ev)
		}
		for _, r := range created {
			ev := dal.NewRecord()
			ev.Set("id", g(r, "id"))
			ev.Set("order_id", g(order, "id"))
			createdEvents = append(createdEvents, ev)
		}
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-reservation-item-deleted", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.deleted", Data: deletedEvents})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-reservation-item-created", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.created", Data: createdEvents})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		pc := dal.NewRecord()
		pc.Set("order_id", g(order, "id"))
		if _, err := CreateOrUpdateOrderPaymentCollectionWorkflow.RunAsStep(e, pc); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("actions", g(orderChange, "actions"))
		ev.Set("no_notification", g(orderChange, "no_notification"))
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order-edit.confirmed", Data: ev}); err != nil {
			return nil, err
		}
		return preview, nil
	}))

var cancelBeginOrderEditValidationStep = editValidationStep("validate-cancel-begin-order-edit")

// CancelBeginOrderEditWorkflow mirrors cancelBeginOrderEditWorkflow.
var CancelBeginOrderEditWorkflow = composer.CreateWorkflow("cancel-begin-order-edit", withOrderLock(func(orderID any) any { return orderID },
	func(e *composer.Execution, orderID any) (any, error) {
		order, err := queryOrderForEdit(e, "order-query", orderID)
		if err != nil {
			return nil, err
		}
		orderChange, err := editActiveChange(e, orderID, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := cancelBeginOrderEditValidationStep.Run(e, changeValidationInput{Order: order, OrderChange: orderChange}); err != nil {
			return nil, err
		}
		shippingToRemove := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == "SHIPPING_ADD" {
				shippingToRemove = append(shippingToRemove, g(a, "id"))
			}
		}
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("actions", g(orderChange, "actions"))
		return nil, composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order-edit.canceled", Data: ev})
				return err
			},
			func() error {
				_, err := ordersteps.DeleteOrderChangesStep.Run(e, []any{g(orderChange, "id")})
				return err
			},
			func() error { _, err := ordersteps.DeleteOrderShippingMethodsStep.Run(e, shippingToRemove); return err },
		)
	}))
