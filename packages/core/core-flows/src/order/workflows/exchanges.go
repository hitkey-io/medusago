package workflows

import (
	"fmt"
	"time"

	cartsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/steps"
	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func exchangeChangeValidationStep(name string) *composer.Step[changeValidationInput, any] {
	return composer.CreateStep(name,
		func(ctx composer.StepExecutionContext, in changeValidationInput) (composer.StepResponse[any], error) {
			if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := throwIfIsCancelled(in.OrderReturn, "Exchange"); err != nil {
				return composer.StepResponse[any]{}, err
			}
			if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
				return composer.StepResponse[any]{}, err
			}
			return composer.NewStepResponse[any](nil), nil
		}, nil)
}

func exchangeActiveChange(e *composer.Execution, exchange *dal.Record, fields []string) (*dal.Record, error) {
	return activeOrderChangeFor(e, g(exchange, "order_id"), "exchange_id", g(exchange, "id"), fields)
}

var beginOrderExchangeValidationStep = composer.CreateStep("begin-exchange-order-validation",
	func(ctx composer.StepExecutionContext, order *dal.Record) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// BeginExchangeOrderWorkflow mirrors beginExchangeOrderWorkflow.
var BeginExchangeOrderWorkflow = composer.CreateWorkflow("begin-exchange-order",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		order, err := remoteOne(e, "", "orders", idVars(g(in, "order_id")), []string{"id", "status"}, true)
		if err != nil {
			return nil, err
		}
		if _, err := beginOrderExchangeValidationStep.Run(e, order); err != nil {
			return nil, err
		}
		c := dal.NewRecord()
		c.Set("order_id", g(in, "order_id"))
		setIfPresent(c, in, "metadata", "created_by")
		created, err := ordersteps.CreateOrderExchangesStep.Run(e, []*dal.Record{c})
		if err != nil {
			return nil, err
		}
		ch := dal.NewRecord()
		ch.Set("change_type", "exchange")
		ch.Set("order_id", g(in, "order_id"))
		ch.Set("exchange_id", g(created[0], "id"))
		setIfPresent(ch, in, "created_by", "description", "internal_note")
		return ordersteps.CreateOrderChangeStep.Run(e, ch)
	})

func refreshExchangeShipping(e *composer.Execution, orderChange, exchange *dal.Record) error {
	args := dal.NewRecord()
	args.Set("order_change_id", g(orderChange, "id"))
	args.Set("exchange_id", g(exchange, "id"))
	args.Set("order_id", g(exchange, "order_id"))
	_, err := RefreshExchangeShippingWorkflow.RunAsStep(e, args)
	return err
}

// RefreshExchangeShippingWorkflow mirrors refreshExchangeShippingWorkflow.
var RefreshExchangeShippingWorkflow = composer.CreateWorkflow("refresh-exchange-shipping",
	func(e *composer.Execution, in *dal.Record) (any, error) {
		f := dal.NewRecord()
		f.Set("order_id", g(in, "order_id"))
		f.Set("exchange_id", g(in, "exchange_id"))
		f.Set("status", []any{"pending", "requested"})
		changes, err := queryList(e, "order-change-query", "order_change", f, []string{"id", "status", "order_id", "exchange_id", "return_id", "actions.*"})
		if err != nil {
			return nil, err
		}
		var orderChange *dal.Record
		if len(changes) > 0 {
			orderChange = changes[0]
		}
		actions := recs(g(orderChange, "actions"))
		itemsOf := func(kind string) []any {
			out := []any{}
			for _, a := range actions {
				if g(a, "action") == kind {
					d := rec(g(a, "details"))
					it := dal.NewRecord()
					it.Set("id", g(d, "reference_id"))
					it.Set("quantity", g(d, "quantity"))
					out = append(out, it)
				}
			}
			return out
		}
		var inbound, outbound *dal.Record
		for _, a := range actions {
			if g(a, "action") != "SHIPPING_ADD" {
				continue
			}
			if truthy(g(a, "return_id")) && inbound == nil {
				inbound = a
			}
			if !truthy(g(a, "return_id")) && outbound == nil {
				outbound = a
			}
		}
		if inbound != nil {
			c := dal.NewRecord()
			c.Set("return_id", g(inbound, "return_id"))
			c.Set("return_items", itemsOf("RETURN_ITEM"))
			args := dal.NewRecord()
			args.Set("shipping_method_id", g(inbound, "reference_id"))
			args.Set("order_id", g(orderChange, "order_id"))
			args.Set("action_id", g(inbound, "id"))
			args.Set("context", c)
			if _, err := MaybeRefreshShippingMethodsWorkflow.RunAsStep(e, args); err != nil {
				return nil, err
			}
		}
		if outbound != nil {
			c := dal.NewRecord()
			c.Set("exchange_id", g(outbound, "exchange_id"))
			c.Set("exchange_items", itemsOf("ITEM_ADD"))
			args := dal.NewRecord()
			args.Set("shipping_method_id", g(outbound, "reference_id"))
			args.Set("order_id", g(orderChange, "order_id"))
			args.Set("action_id", g(outbound, "id"))
			args.Set("context", c)
			if _, err := MaybeRefreshShippingMethodsWorkflow.RunAsStep(e, args); err != nil {
				return nil, err
			}
		}
		return nil, nil
	})

var exchangeRequestItemReturnValidationStep = composer.CreateStep("exchange-request-item-return-validation",
	func(ctx composer.StepExecutionContext, in claimReturnValidationInput) (composer.StepResponse[any], error) {
		fail := func(err error) (composer.StepResponse[any], error) { return composer.StepResponse[any]{}, err }
		if err := throwIfIsCancelled(in.Order, "Order"); err != nil {
			return fail(err)
		}
		if err := throwIfIsCancelled(in.OrderClaim, "Exchange"); err != nil {
			return fail(err)
		}
		if err := throwIfIsCancelled(in.OrderReturn, "Return"); err != nil {
			return fail(err)
		}
		if err := ThrowIfOrderChangeIsNotActive(in.OrderChange); err != nil {
			return fail(err)
		}
		if err := throwIfItemsDoesNotExistsInOrder(in.Order, in.Items); err != nil {
			return fail(err)
		}
		if err := throwIfManagedItemsNotStockedAtReturnLocation(in.Order, in.OrderReturn, in.Items); err != nil {
			return fail(err)
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// OrderExchangeItemsInput mirrors the exchange item workflow inputs.
type OrderExchangeItemsInput struct {
	ExchangeID any
	Items      []*dal.Record
	LocationID any
}

// OrderExchangeRequestItemReturnWorkflow mirrors orderExchangeRequestItemReturnWorkflow.
var OrderExchangeRequestItemReturnWorkflow = composer.CreateWorkflow("exchange-request-item-return",
	func(e *composer.Execution, in OrderExchangeItemsInput) (*dal.Record, error) {
		exchange, err := remoteOne(e, "exchange-query", "order_exchange", idVars(in.ExchangeID), []string{"id", "order_id", "return_id", "location_id", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		var existingReturn *dal.Record
		if truthy(g(exchange, "return_id")) {
			if existingReturn, err = remoteOne(e, "return-query", "return", idVars(g(exchange, "return_id")), []string{"id", "status", "order_id", "canceled_at"}, true); err != nil {
				return nil, err
			}
		}
		order, err := queryOrderOne(e, g(exchange, "order_id"), append(append([]string{}, FieldsToComputeAdjustmentsForPreview...), "canceled_at", "status",
			"items.variant.manage_inventory", "items.variant.inventory_items.inventory_item_id", "items.variant.inventory_items.inventory.location_levels.location_id"))
		if err != nil {
			return nil, err
		}
		orderChange, err := exchangeActiveChange(e, exchange, []string{"id", "status", "version", "exchange_id", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		locationID := in.LocationID
		if !truthy(locationID) {
			locationID = nil
			var item *dal.Record
			if len(in.Items) > 0 {
				for _, oi := range recs(g(order, "items")) {
					if g(oi, "id") == g(in.Items[0], "id") {
						item = oi
						break
					}
				}
			}
			if item != nil {
				cartutils.DeepFlatMap(item, "variant.inventory_items.inventory.location_levels", func(c map[string]*dal.Record) {
					if locationID == nil {
						if v := g(c["location_levels"], "location_id"); v != nil {
							locationID = v
						}
					}
				})
			}
		}
		orderReturn := existingReturn
		var createdReturn []*dal.Record
		if !truthy(g(exchange, "return_id")) {
			r := dal.NewRecord()
			r.Set("order_id", g(exchange, "order_id"))
			if locationID != nil {
				r.Set("location_id", locationID)
			}
			r.Set("exchange_id", g(exchange, "id"))
			if createdReturn, err = ordersteps.CreateReturnsStep.Run(e, []*dal.Record{r}); err != nil {
				return nil, err
			}
			if orderReturn == nil && len(createdReturn) > 0 {
				orderReturn = createdReturn[0]
			}
		}
		if len(createdReturn) > 0 {
			u := dal.NewRecord()
			u.Set("id", g(orderChange, "id"))
			u.Set("return_id", g(createdReturn[0], "id"))
			if _, err := ordersteps.UpdateOrderChangesStep.Run(e, []*dal.Record{u}); err != nil {
				return nil, err
			}
		}
		if _, err := exchangeRequestItemReturnValidationStep.Run(e, claimReturnValidationInput{order, orderReturn, exchange, orderChange, in.Items}); err != nil {
			return nil, err
		}
		if !truthy(g(exchange, "return_id")) {
			u := dal.NewRecord()
			u.Set("id", g(exchange, "id"))
			var rid any
			if len(createdReturn) > 0 {
				rid = g(createdReturn[0], "id")
			}
			u.Set("return_id", rid)
			if _, err := ordersteps.UpdateOrderExchangesStep.Run(e, []*dal.Record{u}); err != nil {
				return nil, err
			}
		}
		actions := make([]*dal.Record, 0, len(in.Items))
		for _, item := range in.Items {
			a := dal.NewRecord()
			a.Set("order_change_id", g(orderChange, "id"))
			a.Set("order_id", g(order, "id"))
			a.Set("exchange_id", g(exchange, "id"))
			a.Set("return_id", g(orderReturn, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "RETURN_ITEM")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "return")
			a.Set("reference_id", g(orderReturn, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(item, "id"))
			d.Set("quantity", g(item, "quantity"))
			setIfPresent(d, item, "reason_id", "metadata")
			a.Set("details", d)
			actions = append(actions, a)
		}
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, actions); err != nil {
			return nil, err
		}
		if err := computeAdjustments(e, order, orderChange); err != nil {
			return nil, err
		}
		if err := refreshExchangeShipping(e, orderChange, exchange); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(exchange, "order_id")))
	})

var exchangeAddNewItemValidationStep = exchangeChangeValidationStep("exchange-add-new-item-validation")

// OrderExchangeAddNewItemWorkflow mirrors orderExchangeAddNewItemWorkflow.
var OrderExchangeAddNewItemWorkflow = composer.CreateWorkflow("exchange-add-new-item",
	func(e *composer.Execution, in OrderExchangeItemsInput) (*dal.Record, error) {
		exchange, err := remoteOne(e, "exchange-query", "order_exchange", idVars(in.ExchangeID), []string{"id", "order_id", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(exchange, "order_id"), append(append([]string{}, FieldsToComputeAdjustmentsForPreview...), "status", "canceled_at"))
		if err != nil {
			return nil, err
		}
		orderChange, err := exchangeActiveChange(e, exchange, []string{"id", "status", "version", "exchange_id", "carry_over_promotions"})
		if err != nil {
			return nil, err
		}
		if _, err := exchangeAddNewItemValidationStep.Run(e, changeValidationInput{order, exchange, orderChange}); err != nil {
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
			a.Set("exchange_id", g(exchange, "id"))
			a.Set("version", g(orderChange, "version"))
			a.Set("action", "ITEM_ADD")
			setIfPresent(a, item, "internal_note")
			a.Set("reference", "order_exchange")
			a.Set("reference_id", g(exchange, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", g(lineItems[i], "id"))
			d.Set("quantity", g(item, "quantity"))
			price := g(item, "unit_price")
			if price == nil {
				price = g(lineItems[i], "unit_price")
			}
			d.Set("unit_price", price)
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
		if err := refreshExchangeShipping(e, orderChange, exchange); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(exchange, "order_id")))
	})

var createExchangeShippingMethodValidationStep = exchangeChangeValidationStep("validate-create-exchange-shipping-method")

// CreateExchangeShippingMethodWorkflow mirrors createExchangeShippingMethodWorkflow.
var CreateExchangeShippingMethodWorkflow = composer.CreateWorkflow("create-exchange-shipping-method",
	func(e *composer.Execution, in *dal.Record) (*dal.Record, error) {
		exchange, err := remoteOne(e, "", "order_exchange", idVars(g(in, "exchange_id")), []string{"id", "status", "order_id", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(exchange, "order_id"), []string{"id", "status", "currency_code", "canceled_at", "locale"})
		if err != nil {
			return nil, err
		}
		orderChange, err := exchangeActiveChange(e, exchange, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		isReturn := truthy(g(in, "return_id"))
		kind := "ITEM_ADD"
		if isReturn {
			kind = "RETURN_ITEM"
		}
		items := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == kind {
				d := rec(g(a, "details"))
				it := dal.NewRecord()
				it.Set("id", g(d, "reference_id"))
				it.Set("quantity", g(d, "quantity"))
				items = append(items, it)
			}
		}
		c := dal.NewRecord()
		if isReturn {
			c.Set("return_id", g(in, "return_id"))
			c.Set("return_items", items)
		} else {
			c.Set("exchange_id", g(in, "exchange_id"))
			c.Set("exchange_items", items)
		}
		option, err := FetchShippingOptionForOrderWorkflow.RunAsStep(e, FetchShippingOptionInput{
			OrderID: g(order, "id"), ShippingOptionID: g(in, "shipping_option_id"), CurrencyCode: g(order, "currency_code"), Context: c,
		})
		if err != nil {
			return nil, err
		}
		if option, err = TranslateShippingOption(e, option, g(order, "locale")); err != nil {
			return nil, err
		}
		if _, err := createExchangeShippingMethodValidationStep.Run(e, changeValidationInput{order, exchange, orderChange}); err != nil {
			return nil, err
		}
		customPrice := g(in, "custom_amount")
		method := PrepareShippingMethod("exchange_id", option, orderChange, exchange, in, customPrice)
		created, err := ordersteps.CreateOrderShippingMethodsStep.Run(e, []*dal.Record{method})
		if err != nil {
			return nil, err
		}
		var methodIDs []any
		for _, m := range created {
			methodIDs = append(methodIDs, g(m, "id"))
		}
		if _, err := UpdateOrderTaxLinesWorkflow.RunAsStep(e, UpdateOrderTaxLinesInput{OrderID: g(order, "id"), ShippingMethodIDs: methodIDs, IsReturn: isReturn}); err != nil {
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
		setIfPresent(a, in, "return_id")
		a.Set("exchange_id", g(exchange, "id"))
		if _, err := CreateOrderChangeActionsWorkflow.RunAsStep(e, []*dal.Record{a}); err != nil {
			return nil, err
		}
		return ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
	})

var confirmExchangeRequestValidationStep = exchangeChangeValidationStep("validate-confirm-exchange-request")

var confirmIfExchangeItemsArePresent = composer.CreateStep("confirm-if-exchange-items-are-present",
	func(ctx composer.StepExecutionContext, in struct {
		ExchangeItems []*dal.Record
		ReturnItems   []*dal.Record
	}) (composer.StepResponse[any], error) {
		if len(in.ExchangeItems) > 0 && len(in.ReturnItems) > 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		return composer.StepResponse[any]{}, invalidData("Order exchange request should have at least 1 item inbound and 1 item outbound")
	}, nil)

// ConfirmExchangeRequestInput mirrors ConfirmExchangeRequestWorkflowInput.
type ConfirmExchangeRequestInput struct {
	ExchangeID  any
	ConfirmedBy any
}

// ConfirmExchangeRequestWorkflow mirrors confirmExchangeRequestWorkflow.
var ConfirmExchangeRequestWorkflow = composer.CreateWorkflow("confirm-exchange-request",
	func(e *composer.Execution, in ConfirmExchangeRequestInput) (*dal.Record, error) {
		exchange, err := remoteOne(e, "", "order_exchange", idVars(in.ExchangeID), []string{"id", "status", "order_id", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(exchange, "order_id"), []string{"id", "version", "canceled_at", "items.*", "items.item.id", "shipping_address.*"})
		if err != nil {
			return nil, err
		}
		orderChange, err := exchangeActiveChange(e, exchange, []string{"id", "status", "actions.id", "actions.exchange_id", "actions.return_id",
			"actions.action", "actions.details", "actions.reference", "actions.reference_id", "actions.internal_note"})
		if err != nil {
			return nil, err
		}
		if _, err := confirmExchangeRequestValidationStep.Run(e, changeValidationInput{order, exchange, orderChange}); err != nil {
			return nil, err
		}
		var exchangeActions, returnActions []*dal.Record
		for _, a := range recs(g(orderChange, "actions")) {
			switch g(a, "action") {
			case "RETURN_ITEM":
				returnActions = append(returnActions, a)
			case "ITEM_ADD":
				exchangeActions = append(exchangeActions, a)
			}
		}
		var exchangeID, returnID any
		if len(exchangeActions) > 0 {
			exchangeID = g(exchangeActions[0], "exchange_id")
		}
		if len(returnActions) > 0 {
			returnID = g(returnActions[0], "return_id")
		}
		preview, err := ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(g(order, "id")))
		if err != nil {
			return nil, err
		}
		var createdExchangeItems, createdReturnItems []*dal.Record
		err = composer.Parallelize(
			func() error {
				var err error
				createdExchangeItems, err = ordersteps.CreateOrderExchangeItemsFromActionsStep.Run(e, ordersteps.CreateExchangeItemsFromActionsInput{Changes: exchangeActions, ExchangeID: exchangeID})
				return err
			},
			func() error {
				var err error
				createdReturnItems, err = ordersteps.CreateReturnItemsFromActionsStep.Run(e, ordersteps.CreateReturnItemsFromActionsInput{Changes: returnActions, ReturnID: returnID})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if _, err := confirmIfExchangeItemsArePresent.Run(e, struct {
			ExchangeItems []*dal.Record
			ReturnItems   []*dal.Record
		}{createdExchangeItems, createdReturnItems}); err != nil {
			return nil, err
		}
		if _, err := ordersteps.ConfirmOrderChangesStep.Run(e, ordersteps.ConfirmOrderChangesInput{OrderID: g(order, "id"), Changes: []*dal.Record{orderChange}, ConfirmedBy: in.ConfirmedBy}); err != nil {
			return nil, err
		}
		var createdReturnID any
		if len(createdReturnItems) > 0 {
			createdReturnID = g(createdReturnItems[0], "return_id")
		}
		if truthy(createdReturnID) {
			u := dal.NewRecord()
			u.Set("id", createdReturnID)
			u.Set("status", "requested")
			u.Set("requested_at", time.Now())
			if _, err := ordersteps.UpdateReturnsStep.Run(e, []*dal.Record{u}); err != nil {
				return nil, err
			}
		}
		var returnShippingMethod, exchangeShippingMethod *dal.Record
		for _, sm := range recs(g(preview, "shipping_methods")) {
			for _, a := range recs(g(sm, "actions")) {
				if g(a, "action") != "SHIPPING_ADD" {
					continue
				}
				if g(a, "return_id") != nil && g(a, "return_id") == createdReturnID {
					returnShippingMethod = sm
				} else if g(a, "exchange_id") == g(exchange, "id") {
					exchangeShippingMethod = sm
				}
			}
		}
		if exchangeShippingMethod != nil {
			full, err := remoteOne(e, "exchange-query", "order_exchange", idVars(in.ExchangeID), []string{"id", "version", "canceled_at", "order.sales_channel_id",
				"additional_items.quantity", "additional_items.raw_quantity", "additional_items.item.id", "additional_items.item.variant.manage_inventory",
				"additional_items.item.variant.allow_backorder", "additional_items.item.variant.inventory_items.inventory_item_id",
				"additional_items.item.variant.inventory_items.required_quantity",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.id",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.name",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.id",
				"additional_items.item.variant.inventory_items.inventory.location_levels.stock_locations.sales_channels.name"}, true)
			if err != nil {
				return nil, err
			}
			var variants, items []*dal.Record
			for _, xi := range recs(g(full, "additional_items")) {
				item := rec(g(xi, "item"))
				it := dal.NewRecord()
				it.Set("id", g(item, "id"))
				it.Set("variant_id", g(item, "variant_id"))
				q := g(xi, "raw_quantity")
				if q == nil {
					q = g(xi, "quantity")
				}
				it.Set("quantity", q)
				items = append(items, it)
				variants = append(variants, rec(g(item, "variant")))
			}
			channelID, _ := g(rec(g(full, "order")), "sales_channel_id").(string)
			inv, err := cartutils.PrepareConfirmInventoryInput(cartutils.ConfirmVariantInventoryInput{SalesChannelID: channelID, Variants: variants, Items: items})
			if err != nil {
				return nil, err
			}
			reservations, err := cartsteps.ReserveInventoryStep.Run(e, inv)
			if err != nil {
				return nil, err
			}
			events := []any{}
			for _, r := range reservations {
				ev := dal.NewRecord()
				ev.Set("id", g(r, "id"))
				ev.Set("order_id", g(order, "id"))
				events = append(events, ev)
			}
			if _, err := commonsteps.EmitEventStep.Config("emit-reservation-item-created", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.created", Data: events}); err != nil {
				return nil, err
			}
		}
		if returnShippingMethod != nil && truthy(createdReturnID) {
			option, err := remoteOne(e, "exchange-return-shipping-option", "shipping_options", idVars(g(returnShippingMethod, "shipping_option_id")),
				[]string{"id", "provider_id", "service_zone.fulfillment_set.location.id", "service_zone.fulfillment_set.location.address.*"}, false)
			if err != nil {
				return nil, err
			}
			data := prepareClaimFulfillmentData(order, recs(g(order, "items")), option)
			fulfillment, err := fulfillmentworkflows.CreateReturnFulfillmentWorkflow.RunAsStep(e, data)
			if err != nil {
				return nil, err
			}
			o := dal.NewRecord()
			o.Set("return_id", createdReturnID)
			f := dal.NewRecord()
			f.Set("fulfillment_id", g(fulfillment, "id"))
			link := dal.NewRecord()
			link.Set(utilsmodulessdk.ORDER, o)
			link.Set(utilsmodulessdk.FULFILLMENT, f)
			if _, err := commonsteps.CreateRemoteLinkStep.Config("exchange-return-shipping-fulfillment-link", composer.StepOptions{}).Run(e, []*dal.Record{link}); err != nil {
				return nil, err
			}
		}
		pc := dal.NewRecord()
		pc.Set("order_id", g(order, "id"))
		if _, err := CreateOrUpdateOrderPaymentCollectionWorkflow.RunAsStep(e, pc); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("exchange_id", g(exchange, "id"))
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.exchange_created", Data: ev}); err != nil {
			return nil, err
		}
		return preview, nil
	})

var cancelBeginOrderExchangeValidationStep = exchangeChangeValidationStep("validate-cancel-begin-order-exchange")

// CancelBeginOrderExchangeWorkflow mirrors cancelBeginOrderExchangeWorkflow.
var CancelBeginOrderExchangeWorkflow = composer.CreateWorkflow("cancel-begin-order-exchange",
	func(e *composer.Execution, exchangeID any) (any, error) {
		exchange, err := remoteOne(e, "", "order_exchange", idVars(exchangeID), []string{"id", "status", "order_id", "return_id", "canceled_at"}, true)
		if err != nil {
			return nil, err
		}
		order, err := queryOrderOne(e, g(exchange, "order_id"), []string{"id", "version", "canceled_at"})
		if err != nil {
			return nil, err
		}
		orderChange, err := exchangeActiveChange(e, exchange, []string{"id", "status", "version", "actions.*"})
		if err != nil {
			return nil, err
		}
		if _, err := cancelBeginOrderExchangeValidationStep.Run(e, changeValidationInput{order, exchange, orderChange}); err != nil {
			return nil, err
		}
		shippingToRemove := []any{}
		for _, a := range recs(g(orderChange, "actions")) {
			if g(a, "action") == "SHIPPING_ADD" {
				shippingToRemove = append(shippingToRemove, g(a, "id"))
			}
		}
		return nil, composer.Parallelize(
			func() error {
				_, err := ordersteps.DeleteReturnsStep.Run(e, []any{g(exchange, "return_id")})
				return err
			},
			func() error { _, err := ordersteps.DeleteExchangesStep.Run(e, []any{g(exchange, "id")}); return err },
			func() error {
				_, err := ordersteps.DeleteOrderChangesStep.Run(e, []any{g(orderChange, "id")})
				return err
			},
			func() error { _, err := ordersteps.DeleteOrderShippingMethodsStep.Run(e, shippingToRemove); return err },
		)
	})
