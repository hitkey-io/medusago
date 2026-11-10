package workflows

import (
	"fmt"
	"time"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	inventorysteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func setIfPresent(dst, src *dal.Record, keys ...string) {
	for _, k := range keys {
		if src == nil {
			return
		}
		if v, ok := src.Get(k); ok {
			dst.Set(k, v)
		}
	}
}

func queryReturn(e *composer.Execution, id any, fields []string) (*dal.Record, error) {
	return remoteOne(e, "", "return", idVars(id), fields, true)
}

func queryOrderOne(e *composer.Execution, id any, fields []string) (*dal.Record, error) {
	return remoteOne(e, "order-query", "orders", idVars(id), fields, true)
}

func orderIDString(order *dal.Record) string {
	return fmt.Sprint(g(order, "id"))
}

// BeginReturnInput mirrors BeginOrderReturnWorkflowInput.
type BeginReturnInput = *dal.Record

var beginReturnOrderValidationStep = composer.CreateStep("begin-return-order-validation",
	func(ctx composer.StepExecutionContext, order *dal.Record) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// BeginReturnOrderWorkflow mirrors beginReturnOrderWorkflow.
var BeginReturnOrderWorkflow = composer.CreateWorkflow("begin-return-order",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		order, err := remoteOne(e, "", "orders", idVars(g(in, "order_id")), []string{"id", "status"}, true)
		if err != nil {
			return nil, err
		}
		if _, err := beginReturnOrderValidationStep.Run(e, order); err != nil {
			return nil, err
		}
		r := dal.NewRecord()
		r.Set("order_id", g(in, "order_id"))
		setIfPresent(r, in, "location_id", "metadata", "created_by")
		created, err := ordersteps.CreateReturnsStep.Run(e, []*dal.Record{r})
		if err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("change_type", "return_request")
		c.Set("order_id", g(in, "order_id"))
		c.Set("return_id", g(created[0], "id"))
		setIfPresent(c, in, "created_by", "description", "internal_note")
		return ordersteps.CreateOrderChangeStep.Run(e, c)
	})

type requestValidationInput struct {
	Order       *dal.Record
	OrderReturn *dal.Record
	OrderChange *dal.Record
	Items       []*dal.Record
}

var requestItemReturnValidationStep = composer.CreateStep("request-item-return-validation",
	func(ctx composer.StepExecutionContext, in requestValidationInput) (composer.StepResponse[any], error) {
		fail := func(err error) (composer.StepResponse[any], error) { return composer.StepResponse[any]{}, err }
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return fail(err)
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return fail(err)
		}
		if err := throwIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return fail(err)
		}
		if err := throwIfItemsDoesNotExistsInOrder(in.Order, in.Items); err != nil {
			return fail(err)
		}
		if err := validateReturnReasons(ctx, g(in.Order, "id"), in.Items); err != nil {
			return fail(err)
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// RequestItemReturnInput mirrors RequestItemReturnWorkflowInput.
type RequestItemReturnInput struct {
	ReturnID any
	Items    []*dal.Record
}

// RequestItemReturnWorkflow mirrors requestItemReturnWorkflow.
var RequestItemReturnWorkflow = composer.CreateWorkflow("request-item-return",
	func(e *composer.Execution, in RequestItemReturnInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := activeOrderChangeFor(e, g(orderReturn, "order_id"), "return_id", g(orderReturn, "id"), []string{"id", "status", "order_id", "return_id"})
		if err != nil {
			return nil, err
		}
		if _, err := requestItemReturnValidationStep.Run(e, requestValidationInput{order, orderReturn, orderChange, in.Items}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("return_id", g(orderReturn, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "RETURN_ITEM")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "return")
			a.Set("reference_id", g(orderReturn, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			setIfPresent(d, item, "reason_id")
			d.Set("quantity", g(item, "quantity"))
			setIfPresent(d, item, "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := createOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		args := dal.NewRecord()
		args.Set("order_change_id", g(orderChange, "id"))
		args.Set("return_id", g(orderReturn, "id"))
		args.Set("order_id", g(orderReturn, "order_id"))
		if _, err := RefreshReturnShippingWorkflow.RunAsStep(e, args); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, orderIDString(order))
	})

type changeValidationInput struct {
	Order       *dal.Record
	OrderReturn *dal.Record
	OrderChange *dal.Record
}

func changeValidationStep(name string) *composer.Step[changeValidationInput, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in changeValidationInput) (composer.StepResponse[any], error) {
			if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := throwIfOrderChangeIsNotActive(in.OrderChange); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil), nil
		}, nil)
}

var createReturnShippingMethodValidationStep = changeValidationStep("validate-create-return-shipping-method")

// CreateReturnShippingMethodInput mirrors CreateReturnShippingMethodWorkflowInput.
type CreateReturnShippingMethodInput struct {
	Input *dal.Record
}

func returnItemsOf(orderChange *dal.Record) []any {
	items := []any{}
	for _, a := range recs(g(orderChange, "actions")) {
		if g(a, "action") == "RETURN_ITEM" {
			d := rec(g(a, "details"))
			it := dal.NewRecord()
			it.Set("id", g(d, "reference_id"))
			it.Set("quantity", g(d, "quantity"))
			items = append(items, it)
		}
	}
	return items
}

// CreateReturnShippingMethodWorkflow mirrors createReturnShippingMethodWorkflow.
var CreateReturnShippingMethodWorkflow = composer.CreateWorkflow("create-return-shipping-method",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, g(in, "return_id"), []string{"id", "status", "order_id", "claim_id", "exchange_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "currency_code", "canceled_at", "locale"})
		if err != nil {
			return nil, err
		}
		orderChange, err := activeOrderChangeFor(e, g(orderReturn, "order_id"), "return_id", g(orderReturn, "id"), []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("return_id", g(orderReturn, "id"))
		c.Set("return_items", returnItemsOf(orderChange))
		option, err := FetchShippingOptionForOrderWorkflow.RunAsStep(e, FetchShippingOptionInput{
			OrderID: g(order, "id"), ShippingOptionID: g(in, "shipping_option_id"), CurrencyCode: g(order, "currency_code"), Context: c,
		})
		if err != nil {
			return nil, err
		}
		if _, err := createReturnShippingMethodValidationStep.Run(e, changeValidationInput{order, orderReturn, orderChange}); err != nil {
			return nil, err
		}
		customPrice := g(in, "custom_amount")
		method := prepareShippingMethod("return_id", option, orderChange, orderReturn, in, customPrice)
		created, err := ordersteps.CreateOrderShippingMethodsStep.Run(e, []*dal.Record{method})
		if err != nil {
			return nil, err
		}
		var methodIDs []any
		for _, m := range created {
			methodIDs = append(methodIDs, g(m, "id"))
		}
		if _, err := UpdateOrderTaxLinesWorkflow.RunAsStep(e, UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ShippingMethodIDs: methodIDs, IsReturn: true}); err != nil {
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
		a.Set("return_id", g(orderReturn, "id"))
		setIfPresent(a, in, "claim_id", "exchange_id")
		if _, err := createOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{a}); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, orderIDString(order))
	})

var confirmReturnRequestValidationStep = changeValidationStep("validate-confirm-return-request")

var confirmIfReturnItemsArePresent = composer.CreateStep("confirm-if-return-items-are-present",
	func(ctx composer.StepExecutionContext, items []*dal.Record) (composer.StepResponse[any], error) {
		if len(items) > 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		return composer.StepResponse[any]{}, invalidData("Order return request should have at least 1 item")
	}, nil)

func extractReturnShippingOptionID(orderPreview, orderReturn *dal.Record) any {
	for _, sm := range recs(g(orderPreview, "shipping_methods")) {
		actions := recs(g(sm, "actions"))
		if actions == nil {
			continue
		}
		for _, a := range actions {
			if g(a, "action") == "SHIPPING_ADD" && g(a, "return_id") == g(orderReturn, "id") {
				return g(sm, "shipping_option_id")
			}
		}
	}
	return nil
}

func prepareReturnFulfillmentData(order *dal.Record, items []*dal.Record, option *dal.Record) *dal.Record {
	orderItems := map[any]*dal.Record{}
	for _, i := range recs(g(order, "items")) {
		orderItems[g(i, "id")] = i
	}
	fitems := []any{}
	for _, i := range items {
		oi := orderItems[g(i, "item_id")]
		fi := dal.NewRecord()
		fi.Set("line_item_id", g(i, "item_id"))
		fi.Set("quantity", g(i, "quantity"))
		fi.Set("return_quantity", g(i, "quantity"))
		title := g(oi, "variant_title")
		if title == nil {
			title = g(oi, "title")
		}
		fi.Set("title", title)
		sku := g(oi, "variant_sku")
		if !truthy(sku) {
			sku = ""
		}
		fi.Set("sku", sku)
		barcode := g(oi, "variant_barcode")
		if !truthy(barcode) {
			barcode = ""
		}
		fi.Set("barcode", barcode)
		fitems = append(fitems, fi)
	}
	location := rec(g(rec(g(rec(g(option, "service_zone")), "fulfillment_set")), "location"))
	address := rec(g(location, "address"))
	addr := dal.NewRecord()
	if address != nil {
		for _, k := range address.Keys() {
			if k == "id" {
				continue
			}
			v, _ := address.Get(k)
			addr.Set(k, v)
		}
	}
	in := dal.NewRecord()
	in.Set("location_id", g(location, "id"))
	in.Set("provider_id", g(option, "provider_id"))
	in.Set("shipping_option_id", g(option, "id"))
	in.Set("items", fitems)
	in.Set("delivery_address", addr)
	in.Set("order", order)
	return in
}

// ConfirmReturnRequestInput mirrors ConfirmReturnRequestWorkflowInput.
type ConfirmReturnRequestInput struct {
	ReturnID    any
	ConfirmedBy any
}

// ConfirmReturnRequestWorkflow mirrors confirmReturnRequestWorkflow.
var ConfirmReturnRequestWorkflow = composer.CreateWorkflow("confirm-return-request",
	func(e *composer.Execution, in ConfirmReturnRequestInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "location_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "version", "canceled_at", "items.id", "items.title", "items.variant_title", "items.variant_sku", "items.variant_barcode"})
		if err != nil {
			return nil, err
		}
		orderChange, err := activeOrderChangeFor(e, g(orderReturn, "order_id"), "return_id", g(orderReturn, "id"),
			[]string{"id", "status", "actions.id", "actions.action", "actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"})
		if err != nil {
			return nil, err
		}
		var returnItemActions []*dal.Record
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == "RETURN_ITEM" {
				returnItemActions = append(returnItemActions, a)
			}
		}
		if _, err := confirmReturnRequestValidationStep.Run(e, changeValidationInput{order, orderReturn, orderChange}); err != nil {
			return nil, err
		}
		preview, err := ordersteps.PreviewOrderChangeStep.Run(e, orderIDString(order))
		if err != nil {
			return nil, err
		}
		createdItems, err := ordersteps.CreateReturnItemsFromActionsStep.Run(e, ordersteps.CreateReturnItemsFromActionsInput{ReturnID: g(orderReturn, "id"), Changes: returnItemActions})
		if err != nil {
			return nil, err
		}
		if _, err := confirmIfReturnItemsArePresent.Run(e, createdItems); err != nil {
			return nil, err
		}
		if optionID := extractReturnShippingOptionID(preview, orderReturn); truthy(optionID) {
			option, err := remoteOne(e, "return-shipping-option", "shipping_options", idVars(optionID),
				[]string{"id", "provider_id", "service_zone.fulfillment_set.location.id", "service_zone.fulfillment_set.location.address.*"}, false)
			if err != nil {
				return nil, err
			}
			data := prepareReturnFulfillmentData(order, createdItems, option)
			fulfillment, err := fulfillmentworkflows.CreateReturnFulfillmentWorkflow.RunAsStep(e, data)
			if err != nil {
				return nil, err
			}
			o := dal.NewRecord()
			o.Set("return_id", g(orderReturn, "id"))
			f := dal.NewRecord()
			f.Set("fulfillment_id", g(fulfillment, "id"))
			link := dal.NewRecord()
			link.Set(utilsmodulessdk.ORDER, o)
			link.Set(utilsmodulessdk.FULFILLMENT, f)
			if _, err := commonsteps.CreateRemoteLinkStep.Run(e, []*dal.Record{link}); err != nil {
				return nil, err
			}
		}
		u := dal.NewRecord()
		u.Set("id", g(orderReturn, "id"))
		u.Set("status", "requested")
		u.Set("requested_at", time.Now())
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("return_id", g(orderReturn, "id"))
		err = composer.Parallelize(
			func() error { _, err := ordersteps.UpdateReturnsStep.Run(e, []*dal.Record{u}); return err },
			func() error {
				_, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}, ConfirmedBy: in.ConfirmedBy})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.return_requested", Data: ev})
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
		return preview, nil
	})

var beginReceiveReturnValidationStep = composer.CreateStep("begin-receive-return-validation",
	func(ctx composer.StepExecutionContext, in changeValidationInput) (composer.StepResponse[any], error) {
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// BeginReceiveReturnWorkflow mirrors beginReceiveReturnWorkflow.
var BeginReceiveReturnWorkflow = composer.CreateWorkflow("begin-receive-return",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, g(in, "return_id"), []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "version", "status", "canceled_at"})
		if err != nil {
			return nil, err
		}
		if _, err := beginReceiveReturnValidationStep.Run(e, changeValidationInput{Order: order, OrderReturn: orderReturn}); err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("change_type", "return_receive")
		c.Set("order_id", g(order, "id"))
		c.Set("return_id", g(orderReturn, "id"))
		setIfPresent(c, in, "created_by", "description", "internal_note")
		return ordersteps.CreateOrderChangeStep.Run(e, c)
	})

var receiveItemReturnRequestValidationStep = composer.CreateStep("receive-item-return-request-validation",
	func(ctx composer.StepExecutionContext, in requestValidationInput) (composer.StepResponse[any], error) {
		fail := func(err error) (composer.StepResponse[any], error) { return composer.StepResponse[any]{}, err }
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return fail(err)
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return fail(err)
		}
		if err := throwIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return fail(err)
		}
		if err := throwIfItemsDoesNotExistsInReturn(in.OrderReturn, in.Items); err != nil {
			return fail(err)
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// ReceiveItemReturnRequestWorkflow mirrors receiveItemReturnRequestWorkflow.
var ReceiveItemReturnRequestWorkflow = composer.CreateWorkflow("receive-item-return-request",
	func(e *composer.Execution, in RequestItemReturnInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "canceled_at", "items.*"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "status", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := activeOrderChangeFor(e, g(orderReturn, "order_id"), "return_id", g(orderReturn, "id"), []string{"id", "status", "order_id", "return_id"})
		if err != nil {
			return nil, err
		}
		if _, err := receiveItemReturnRequestValidationStep.Run(e, requestValidationInput{order, orderReturn, orderChange, in.Items}); err != nil {
			return nil, err
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("return_id", g(orderReturn, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "RECEIVE_RETURN_ITEM")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "return")
			a.Set("reference_id", g(orderReturn, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			d.Set("quantity", g(item, "quantity"))
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := createOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, orderIDString(order))
	})

var confirmReceiveReturnValidationStep = changeValidationStep("validate-confirm-return-receive")

// prepareReturnInventoryUpdate mirrors prepareInventoryUpdate of
// confirmReturnReceiveWorkflow.
func prepareReturnInventoryUpdate(orderReturn *dal.Record, variantOrder []any, returnedQuantityMap map[any]any) ([]*dal.Record, error) {
	type invItem struct {
		variantID, inventoryItemID, requiredQuantity any
	}
	var ordered []any
	byID := map[any]invItem{}
	hasManaged, hasLocation := false, false
	locationID := g(orderReturn, "location_id")
	for _, ri := range recs(g(orderReturn, "items")) {
		variant := rec(g(rec(g(ri, "item")), "variant"))
		if variant == nil {
			continue
		}
		visit := func(ii *dal.Record, level *dal.Record) {
			if !truthy(g(variant, "manage_inventory")) {
				return
			}
			hasManaged = true
			if g(level, "location_id") != locationID {
				return
			}
			hasLocation = true
			if ii == nil {
				return
			}
			id := g(ii, "inventory_item_id")
			if _, ok := byID[id]; !ok {
				ordered = append(ordered, id)
				byID[id] = invItem{g(ii, "variant_id"), id, g(ii, "required_quantity")}
			}
		}
		iis := recs(g(variant, "inventory_items"))
		if len(iis) == 0 {
			visit(nil, nil)
		}
		for _, ii := range iis {
			levels := recs(g(rec(g(ii, "inventory")), "location_levels"))
			if len(levels) == 0 {
				visit(ii, nil)
			}
			for _, l := range levels {
				visit(ii, l)
			}
		}
	}
	if hasManaged && !hasLocation {
		return nil, fmt.Errorf("Cannot receive the Return at location %v", locationID)
	}
	out := []*dal.Record{}
	for _, variantID := range variantOrder {
		for _, id := range ordered {
			it := byID[id]
			if it.variantID != variantID {
				continue
			}
			a := dal.NewRecord()
			a.Set("inventory_item_id", it.inventoryItemID)
			a.Set("location_id", locationID)
			a.Set("adjustment", totals.Mult(returnedQuantityMap[variantID], it.requiredQuantity))
			out = append(out, a)
		}
	}
	return out, nil
}

// ConfirmReturnReceiveWorkflow mirrors confirmReturnReceiveWorkflow.
var ConfirmReturnReceiveWorkflow = composer.CreateWorkflow("confirm-return-receive",
	func(e *composer.Execution, in ConfirmReturnRequestInput) (*dal.Record, error) {
		orderReturn, err := queryReturn(e, in.ReturnID, []string{"id", "status", "order_id", "location_id", "canceled_at", "items.*", "items.item.variant_id",
			"items.item.variant.manage_inventory", "items.item.variant.inventory_items.inventory_item_id", "items.item.variant.inventory_items.required_quantity",
			"items.item.variant.inventory_items.inventory.location_levels.location_id"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "version", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := activeOrderChangeFor(e, g(orderReturn, "order_id"), "return_id", g(orderReturn, "id"),
			[]string{"id", "status", "actions.id", "actions.action", "actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"})
		if err != nil {
			return nil, err
		}

		retItems := recs(g(orderReturn, "items"))
		returnedQuantityMap := map[any]any{}
		var variantOrder []any
		var received []*dal.Record
		for _, act := range recs(g(orderChange, "actions")) {
			action := g(act, "action")
			if action != "RECEIVE_RETURN_ITEM" && action != "RECEIVE_DAMAGED_RETURN_ITEM" {
				continue
			}
			received = append(received, act)
			if action != "RECEIVE_RETURN_ITEM" {
				continue
			}
			d := rec(g(act, "details"))
			var variantID any
			for _, ri := range retItems {
				if g(ri, "item_id") == g(d, "reference_id") {
					variantID = g(rec(g(ri, "item")), "variant_id")
					break
				}
			}
			if !truthy(variantID) {
				continue
			}
			cur, ok := returnedQuantityMap[variantID]
			if !ok {
				cur = 0
				variantOrder = append(variantOrder, variantID)
			}
			returnedQuantityMap[variantID] = totals.Add(cur, g(d, "quantity"))
		}
		itemMap := map[any]any{}
		for _, ri := range retItems {
			itemMap[g(ri, "item_id")] = g(ri, "id")
		}
		var updateOrder []any
		itemUpdates := map[any]*dal.Record{}
		for _, act := range received {
			d := rec(g(act, "details"))
			itemID := g(d, "reference_id")
			if u, ok := itemUpdates[itemID]; ok {
				u.Set("received_quantity", totals.Add(g(u, "received_quantity"), g(d, "quantity")))
				if g(act, "action") == "RECEIVE_DAMAGED_RETURN_ITEM" {
					u.Set("damaged_quantity", totals.Add(g(u, "damaged_quantity"), g(d, "quantity")))
				}
				continue
			}
			u := dal.NewRecord()
			u.Set("id", itemMap[itemID])
			u.Set("received_quantity", g(d, "quantity"))
			if g(act, "action") == "RECEIVE_DAMAGED_RETURN_ITEM" {
				u.Set("damaged_quantity", g(d, "quantity"))
			} else {
				u.Set("damaged_quantity", 0)
			}
			itemUpdates[itemID] = u
			updateOrder = append(updateOrder, itemID)
		}
		all := true
		for _, ri := range retItems {
			recv := g(ri, "received_quantity")
			if u, ok := itemUpdates[g(ri, "item_id")]; ok {
				recv = g(u, "received_quantity")
			}
			if !totals.Eq(recv, g(ri, "quantity")) {
				all = false
			}
		}
		updateReturn := dal.NewRecord()
		updateReturn.Set("id", g(orderReturn, "id"))
		if all {
			updateReturn.Set("status", "received")
			updateReturn.Set("received_at", time.Now())
		} else {
			updateReturn.Set("status", "partially_received")
		}
		updateItems := make([]*dal.Record, 0, len(updateOrder))
		for _, id := range updateOrder {
			updateItems = append(updateItems, itemUpdates[id])
		}

		adjustments, err := prepareReturnInventoryUpdate(orderReturn, variantOrder, returnedQuantityMap)
		if err != nil {
			return nil, err
		}
		if _, err := confirmReceiveReturnValidationStep.Run(e, changeValidationInput{order, orderReturn, orderChange}); err != nil {
			return nil, err
		}
		var adjusted []*dal.Record
		err = composer.Parallelize(
			func() error { _, err := ordersteps.UpdateReturnsStep.Run(e, []*dal.Record{updateReturn}); return err },
			func() error { _, err := ordersteps.UpdateReturnItemsStep.Run(e, updateItems); return err },
			func() error {
				_, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}, ConfirmedBy: in.ConfirmedBy})
				return err
			},
			func() error {
				var err error
				adjusted, err = inventorysteps.AdjustInventoryLevelsStep.Run(e, adjustments)
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		levelEvents := []any{}
		for _, l := range adjusted {
			ev := dal.NewRecord()
			ev.Set("id", g(l, "id"))
			ev.Set("order_id", g(order, "id"))
			levelEvents = append(levelEvents, ev)
		}
		pc := dal.NewRecord()
		pc.Set("order_id", g(order, "id"))
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("return_id", g(orderReturn, "id"))
		err = composer.Parallelize(
			func() error { _, err := CreateOrUpdateOrderPaymentCollectionWorkflow.RunAsStep(e, pc); return err },
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.return_received", Data: ev})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-inventory-level-updated", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.updated", Data: levelEvents})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, orderIDString(order))
	})

var cancelRequestReturnValidationStep = changeValidationStep("validate-cancel-return-shipping-method")

// CancelReturnRequestWorkflow mirrors cancelReturnRequestWorkflow.
var CancelReturnRequestWorkflow = composer.CreateWorkflow("cancel-return-request",
	func(e *composer.Execution, returnID any) (any, error) {
		orderReturn, err := queryReturn(e, returnID, []string{"id", "status", "order_id", "canceled_at"})
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(orderReturn, "order_id"), []string{"id", "version", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := activeOrderChangeFor(e, g(orderReturn, "order_id"), "return_id", g(orderReturn, "id"), []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := cancelRequestReturnValidationStep.Run(e, changeValidationInput{order, orderReturn, orderChange}); err != nil {
			return nil, err
		}
		// Upstream passes the SHIPPING_ADD action ids (not the shipping method ids).
		shippingToRemove := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == "SHIPPING_ADD" {
				shippingToRemove = append(shippingToRemove, g(a, "id"))
			}
		}
		return nil, composer.Parallelize(
			func() error { _, err := ordersteps.DeleteReturnsStep.Run(e, []any{g(orderReturn, "id")}); return err },
			func() error {
				_, err := ordersteps.DeleteOrderChangesStep.Run(e, []any{g(orderChange, "id")})
				return err
			},
			func() error { _, err := ordersteps.DeleteOrderShippingMethodsStep.Run(e, shippingToRemove); return err },
		)
	})
