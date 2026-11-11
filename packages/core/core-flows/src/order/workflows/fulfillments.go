package workflows

import (
	"fmt"
	"strings"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	inventorysteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	reservationsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/reservation/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func g(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// setDefined copies key from src when it is given: grouped events are
// released through a JSON round trip upstream, undefined keys are dropped.
func setDefined(dst *dal.Record, key string, src *dal.Record) {
	if src == nil {
		return
	}
	if v, ok := src.Get(key); ok {
		dst.Set(key, v)
	}
}

func rec(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

func recs(v any) []*dal.Record {
	switch l := v.(type) {
	case []*dal.Record:
		return l
	case []any:
		var out []*dal.Record
		for _, x := range l {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func toAnyList(list []*dal.Record) []any {
	out := make([]any, len(list))
	for i, r := range list {
		out[i] = r
	}
	return out
}

func invalidData(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	}
	return true
}

// throwIfOrderIsCancelled mirrors throwIfOrderIsCancelled.
func throwIfOrderIsCancelled(order *dal.Record) error {
	if g(order, "status") == "canceled" {
		return invalidData(fmt.Sprintf("Order with id %v has been canceled.", g(order, "id")))
	}
	return nil
}

// throwIfItemsDoesNotExistsInOrder mirrors throwIfItemsDoesNotExistsInOrder.
func throwIfItemsDoesNotExistsInOrder(order *dal.Record, inputItems []*dal.Record) error {
	ids := map[any]bool{}
	for _, i := range recs(g(order, "items")) {
		ids[g(i, "id")] = true
	}
	var diff []string
	for _, i := range inputItems {
		if !ids[g(i, "id")] {
			diff = append(diff, fmt.Sprint(g(i, "id")))
		}
	}
	if len(diff) > 0 {
		return invalidData(fmt.Sprintf("Items with ids %s does not exist in order with id %v.", strings.Join(diff, ","), g(order, "id")))
	}
	return nil
}

func throwIfItemsAreNotGroupedByShippingRequirement(order *dal.Record, inputItems []*dal.Record) error {
	items := map[any]*dal.Record{}
	for _, i := range recs(g(order, "items")) {
		items[g(i, "id")] = i
	}
	with, without := 0, 0
	for _, in := range inputItems {
		if truthy(g(items[g(in, "id")], "requires_shipping")) {
			with++
		} else {
			without++
		}
	}
	if with > 0 && without > 0 {
		return invalidData("Fulfillment can only be created entirely with items with shipping or items without shipping. Split this request into 2 fulfillments.")
	}
	return nil
}

func queryOrder(e *composer.Execution, name, id string, fields []string) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	isList := false
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: "order", Fields: fields, Filters: f},
		Options:    joiner.Options{ThrowIfKeyNotFound: true},
		IsList:     &isList,
	})
	if err != nil {
		return nil, err
	}
	return rec(res.Data), nil
}

func reservationsFor(e *composer.Execution, lineItemIDs []any) ([]*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("line_item_id", lineItemIDs)
	v := dal.NewRecord()
	v.Set("filters", f)
	res, err := commonsteps.UseRemoteQueryStep.Config("get-reservations", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
		EntryPoint: "reservations", Fields: []string{"id", "line_item_id", "quantity", "inventory_item_id", "location_id"}, Variables: v,
	})
	if err != nil {
		return nil, err
	}
	return recs(res.Data), nil
}

func buildReservationsMap(reservations []*dal.Record) map[any][]*dal.Record {
	m := map[any][]*dal.Record{}
	for _, r := range reservations {
		m[g(r, "line_item_id")] = append(m[g(r, "line_item_id")], r)
	}
	return m
}

func inventoryItemFor(orderItem *dal.Record, inventoryItemID any) *dal.Record {
	for _, ii := range recs(g(rec(g(orderItem, "variant")), "inventory_items")) {
		if g(rec(g(ii, "inventory")), "id") == inventoryItemID {
			return ii
		}
	}
	return nil
}

// CreateOrderFulfillmentInput mirrors CreateOrderFulfillmentWorkflowInput.
type CreateOrderFulfillmentInput struct {
	OrderID string
	Input   *dal.Record
}

var createFulfillmentOrderFields = []string{"id", "display_id", "custom_display_id", "status", "customer_id", "customer.*", "sales_channel_id", "sales_channel.*",
	"region_id", "region.*", "currency_code", "items.*", "items.variant.manage_inventory", "items.variant.allow_backorder", "items.variant.product.id",
	"items.variant.product.shipping_profile.id", "items.variant.weight", "items.variant.length", "items.variant.height", "items.variant.width",
	"items.variant.material", "items.variant_title", "items.variant.upc", "items.variant.sku", "items.variant.barcode", "items.variant.hs_code",
	"items.variant.origin_country", "items.variant.product.origin_country", "items.variant.product.hs_code", "items.variant.product.mid_code",
	"items.variant.product.material", "items.tax_lines.rate", "metadata", "subtotal", "discount_total", "tax_total", "item_total", "shipping_total",
	"total", "created_at", "items.variant.inventory_items.required_quantity", "items.variant.inventory_items.inventory.id",
	"items.variant.inventory_items.inventory.title", "items.variant.inventory_items.inventory.sku", "shipping_address.*", "shipping_methods.id",
	"shipping_methods.shipping_option_id", "shipping_methods.data", "shipping_methods.amount"}

var createFulfillmentValidateOrder = composer.CreateStep("create-fulfillment-validate-order",
	func(ctx composer.StepExecutionContext, in struct {
		Order *dal.Record
		Items []*dal.Record
	}) (composer.StepResponse[any], error) {
		if len(in.Items) == 0 {
			return composer.StepResponse[any]{}, invalidData("No items to fulfill")
		}
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfItemsDoesNotExistsInOrder(in.Order, in.Items); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfItemsAreNotGroupedByShippingRequirement(in.Order, in.Items); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateOrderFulfillmentWorkflow mirrors createOrderFulfillmentWorkflow.
var CreateOrderFulfillmentWorkflow = composer.CreateWorkflow("create-order-fulfillment",
	func(e *composer.Execution, in CreateOrderFulfillmentInput) (*dal.Record, error) {
		input := in.Input
		order, err := queryOrder(e, "get-order", in.OrderID, createFulfillmentOrderFields)
		if err != nil {
			return nil, err
		}
		inputItems := recs(g(input, "items"))
		if _, err := createFulfillmentValidateOrder.Run(e, struct {
			Order *dal.Record
			Items []*dal.Record
		}{order, inputItems}); err != nil {
			return nil, err
		}
		inputItemsMap := map[any]*dal.Record{}
		for _, it := range inputItems {
			inputItemsMap[g(it, "id")] = it
		}
		orderItems := recs(g(order, "items"))
		if il := recs(g(input, "items_list")); il != nil {
			orderItems = il
		}
		shippingOptionID := g(input, "shipping_option_id")
		if shippingOptionID == nil {
			if sms := recs(g(order, "shipping_methods")); len(sms) > 0 {
				shippingOptionID = g(sms[0], "shipping_option_id")
			}
		}
		var shippingMethodData any
		for _, sm := range recs(g(order, "shipping_methods")) {
			if g(sm, "shipping_option_id") == shippingOptionID {
				shippingMethodData = g(sm, "data")
				break
			}
		}
		sov := dal.NewRecord()
		sov.Set("id", shippingOptionID)
		list := false
		soRes, err := commonsteps.UseRemoteQueryStep.Config("get-shipping-option", composer.StepOptions{}).Run(e, commonsteps.RemoteStepInput{
			EntryPoint: "shipping_options", Fields: []string{"id", "provider_id", "service_zone.fulfillment_set.location.id", "shipping_profile_id"}, Variables: sov, List: &list,
		})
		if err != nil {
			return nil, err
		}
		var shippingOption *dal.Record
		if len(soRes.Data) > 0 {
			shippingOption = rec(soRes.Data[0])
		}
		var lineItemIDs []any
		for _, i := range orderItems {
			if _, ok := inputItemsMap[g(i, "id")]; ok {
				lineItemIDs = append(lineItemIDs, g(i, "id"))
			}
		}
		reservations, err := reservationsFor(e, lineItemIDs)
		if err != nil {
			return nil, err
		}
		fulfillmentInput, err := prepareFulfillmentData(order, input, shippingOption, shippingMethodData, reservations, orderItems)
		if err != nil {
			return nil, err
		}
		fulfillment, err := fulfillmentworkflows.CreateFulfillmentWorkflow.RunAsStep(e, fulfillmentInput)
		if err != nil {
			return nil, err
		}
		registerItems := []any{}
		for _, i := range orderItems {
			qty := g(i, "quantity")
			if in, ok := inputItemsMap[g(i, "id")]; ok && g(in, "quantity") != nil {
				qty = g(in, "quantity")
			}
			r := dal.NewRecord()
			r.Set("id", g(i, "id"))
			r.Set("quantity", qty)
			registerItems = append(registerItems, r)
		}
		// itemsList: input.items ?? input.items_list
		if inputItems != nil {
			registerItems = []any{}
			for _, i := range inputItems {
				r := dal.NewRecord()
				r.Set("id", g(i, "id"))
				qty := g(i, "quantity")
				if in, ok := inputItemsMap[g(i, "id")]; ok && g(in, "quantity") != nil {
					qty = g(in, "quantity")
				}
				r.Set("quantity", qty)
				registerItems = append(registerItems, r)
			}
		}
		register := dal.NewRecord()
		register.Set("order_id", g(order, "id"))
		register.Set("reference", utilsmodulessdk.FULFILLMENT)
		register.Set("reference_id", g(fulfillment, "id"))
		register.Set("created_by", g(input, "created_by"))
		register.Set("items", registerItems)

		o := dal.NewRecord()
		o.Set("order_id", in.OrderID)
		f := dal.NewRecord()
		f.Set("fulfillment_id", g(fulfillment, "id"))
		link := dal.NewRecord()
		link.Set(utilsmodulessdk.ORDER, o)
		link.Set(utilsmodulessdk.FULFILLMENT, f)

		toDelete, toUpdate, adjustments, err := prepareFulfillmentInventoryUpdate(reservations, order, input, inputItemsMap, orderItems)
		if err != nil {
			return nil, err
		}
		adjustedLevels, err := inventorysteps.AdjustInventoryLevelsStep.Run(e, adjustments)
		if err != nil {
			return nil, err
		}
		created := dal.NewRecord()
		created.Set("order_id", in.OrderID)
		created.Set("fulfillment_id", g(fulfillment, "id"))
		setDefined(created, "no_notification", input)
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.fulfillment_created", Data: created})
				return err
			},
			func() error { _, err := ordersteps.RegisterOrderFulfillmentStep.Run(e, register); return err },
			func() error { _, err := commonsteps.CreateRemoteLinkStep.Run(e, []*dal.Record{link}); return err },
			func() error { _, err := reservationsteps.UpdateReservationsStep.Run(e, toUpdate); return err },
			func() error { _, err := reservationsteps.DeleteReservationsStep.Run(e, toDelete); return err },
		)
		if err != nil {
			return nil, err
		}
		levelEvents := []any{}
		for _, l := range adjustedLevels {
			ev := dal.NewRecord()
			ev.Set("id", g(l, "id"))
			ev.Set("order_id", in.OrderID)
			levelEvents = append(levelEvents, ev)
		}
		updEvents, delEvents := []any{}, []any{}
		for _, r := range toUpdate {
			ev := dal.NewRecord()
			ev.Set("id", g(r, "id"))
			ev.Set("order_id", in.OrderID)
			updEvents = append(updEvents, ev)
		}
		for _, id := range toDelete {
			ev := dal.NewRecord()
			ev.Set("id", id)
			ev.Set("order_id", in.OrderID)
			delEvents = append(delEvents, ev)
		}
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-inventory-level-updated", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.updated", Data: levelEvents})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-reservation-item-updated", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.updated", Data: updEvents})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-reservation-item-deleted", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.deleted", Data: delEvents})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("fulfillment", fulfillment)
		hook.Set("additional_data", g(input, "additional_data"))
		if err := composer.CreateHook(e, "fulfillmentCreated", hook); err != nil {
			return nil, err
		}
		return fulfillment, nil
	})

func prepareFulfillmentData(order, input, shippingOption *dal.Record, shippingMethodData any, reservations, orderItems []*dal.Record) (*dal.Record, error) {
	inputItems := recs(g(input, "items"))
	itemsMap := map[any]*dal.Record{}
	for _, i := range orderItems {
		itemsMap[g(i, "id")] = i
	}
	reservationMap := buildReservationsMap(reservations)
	someRequireShipping := true
	if len(inputItems) > 0 {
		someRequireShipping = false
		for _, it := range inputItems {
			if truthy(g(itemsMap[g(it, "id")], "requires_shipping")) {
				someRequireShipping = true
				break
			}
		}
	}
	or := func(a, b any) any {
		if truthy(a) {
			return a
		}
		return b
	}
	var items []any
	for _, i := range inputItems {
		orderItem := itemsMap[g(i, "id")]
		res := reservationMap[g(i, "id")]
		variant := rec(g(orderItem, "variant"))
		product := rec(g(variant, "product"))
		if truthy(g(orderItem, "requires_shipping")) && product != nil && g(rec(g(product, "shipping_profile")), "id") != g(shippingOption, "shipping_profile_id") {
			return nil, invalidData(fmt.Sprintf("Shipping profile %v does not match the shipping profile of the order item %v", g(shippingOption, "shipping_profile_id"), g(orderItem, "id")))
		}
		if len(res) == 0 {
			fi := dal.NewRecord()
			fi.Set("line_item_id", g(i, "id"))
			fi.Set("inventory_item_id", nil)
			fi.Set("quantity", g(i, "quantity"))
			title := g(orderItem, "variant_title")
			if title == nil {
				title = g(orderItem, "title")
			}
			fi.Set("title", title)
			fi.Set("sku", or(g(orderItem, "variant_sku"), ""))
			fi.Set("barcode", or(g(orderItem, "variant_barcode"), ""))
			items = append(items, fi)
			continue
		}
		for _, r := range res {
			iItem := inventoryItemFor(orderItem, g(r, "inventory_item_id"))
			inv := rec(g(iItem, "inventory"))
			required := g(iItem, "required_quantity")
			if required == nil {
				required = 1
			}
			fi := dal.NewRecord()
			fi.Set("line_item_id", g(i, "id"))
			fi.Set("inventory_item_id", g(r, "inventory_item_id"))
			fi.Set("quantity", totals.Mult(required, g(i, "quantity")))
			fi.Set("title", or(g(inv, "title"), or(g(orderItem, "variant_title"), g(orderItem, "title"))))
			fi.Set("sku", or(g(inv, "sku"), or(g(orderItem, "variant_sku"), "")))
			fi.Set("barcode", or(g(orderItem, "variant_barcode"), ""))
			items = append(items, fi)
		}
	}
	locationID := g(input, "location_id")
	if !truthy(locationID) {
		locationID = g(rec(g(rec(g(rec(g(shippingOption, "service_zone")), "fulfillment_set")), "location")), "id")
	}
	if !truthy(locationID) {
		return nil, invalidData(fmt.Sprintf("Cannot create fulfillment without stock location, either provide a location or you should link the shipping option %v to a stock location.", g(shippingOption, "id")))
	}
	address := dal.NewRecord()
	if sa := rec(g(order, "shipping_address")); sa != nil {
		for _, k := range sa.Keys() {
			if k == "id" {
				continue
			}
			v, _ := sa.Get(k)
			address.Set(k, v)
		}
		sa.Delete("id")
	}
	if da := rec(g(input, "delivery_address")); da != nil {
		for _, k := range da.Keys() {
			v, _ := da.Get(k)
			address.Set(k, v)
		}
	}
	labels := g(input, "labels")
	if labels == nil {
		labels = []any{}
	}
	out := dal.NewRecord()
	out.Set("location_id", locationID)
	out.Set("provider_id", g(shippingOption, "provider_id"))
	out.Set("shipping_option_id", g(shippingOption, "id"))
	out.Set("order", order)
	out.Set("data", shippingMethodData)
	if items == nil {
		items = []any{}
	}
	out.Set("items", items)
	out.Set("requires_shipping", someRequireShipping)
	out.Set("labels", labels)
	out.Set("delivery_address", address)
	out.Set("created_by", g(input, "created_by"))
	out.Set("packed_at", dal.Now())
	out.Set("metadata", g(input, "metadata"))
	out.Set("additional_data", g(input, "additional_data"))
	return out, nil
}

func prepareFulfillmentInventoryUpdate(reservations []*dal.Record, order, input *dal.Record, inputItemsMap map[any]*dal.Record, orderItems []*dal.Record) ([]any, []*dal.Record, []*dal.Record, error) {
	toDelete := []any{}
	toUpdate := []*dal.Record{}
	adjustments := []*dal.Record{}
	reservationMap := buildReservationsMap(reservations)
	for _, item := range orderItems {
		in, ok := inputItemsMap[g(item, "id")]
		if !ok {
			continue
		}
		res := reservationMap[g(item, "id")]
		if len(res) == 0 {
			if truthy(g(rec(g(item, "variant")), "manage_inventory")) {
				return nil, nil, nil, invalidData(fmt.Sprintf("No stock reservation found for item %v - %v (%v)", g(item, "id"), g(item, "title"), g(item, "variant_title")))
			}
			continue
		}
		inputQuantity := g(in, "quantity")
		if inputQuantity == nil {
			inputQuantity = g(item, "quantity")
		}
		for _, r := range res {
			iItem := inventoryItemFor(item, g(r, "inventory_item_id"))
			required := g(iItem, "required_quantity")
			if required == nil {
				required = 1
			}
			adj := totals.Mult(inputQuantity, required)
			remaining := totals.Sub(g(r, "quantity"), adj)
			if totals.Lt(remaining, 0) {
				return nil, nil, nil, invalidData(fmt.Sprintf("Quantity to fulfill exceeds the reserved quantity for the item: %v", g(item, "id")))
			}
			loc := g(input, "location_id")
			if loc == nil {
				loc = g(r, "location_id")
			}
			a := dal.NewRecord()
			a.Set("inventory_item_id", g(r, "inventory_item_id"))
			a.Set("location_id", loc)
			a.Set("adjustment", totals.Mult(adj, -1))
			adjustments = append(adjustments, a)
			if totals.Eq(remaining, 0) {
				toDelete = append(toDelete, g(r, "id"))
			} else {
				u := dal.NewRecord()
				u.Set("id", g(r, "id"))
				u.Set("quantity", remaining)
				u.Set("location_id", loc)
				toUpdate = append(toUpdate, u)
			}
		}
	}
	return toDelete, toUpdate, adjustments, nil
}

// orderFulfillmentItems computes the line item quantities of a fulfillment
// (prepareRegisterShipmentData / prepareRegisterDeliveryData /
// prepareCancelOrderFulfillmentData).
func orderFulfillmentItems(order, fulfillment *dal.Record, requireManaged, skipMissing bool) []any {
	var lineItemIDs []any
	seen := map[any]bool{}
	fitems := recs(g(fulfillment, "items"))
	for _, i := range fitems {
		if !seen[g(i, "line_item_id")] {
			seen[g(i, "line_item_id")] = true
			lineItemIDs = append(lineItemIDs, g(i, "line_item_id"))
		}
	}
	out := []any{}
	for _, lineItemID := range lineItemIDs {
		var orderItem *dal.Record
		for _, i := range recs(g(order, "items")) {
			if g(i, "id") == lineItemID {
				orderItem = i
			}
		}
		variant := rec(g(orderItem, "variant"))
		iitems := recs(g(variant, "inventory_items"))
		if requireManaged && !truthy(g(variant, "manage_inventory")) {
			iitems = nil
		}
		var fitem *dal.Record
		for _, i := range fitems {
			if g(i, "line_item_id") == lineItemID {
				fitem = i
				break
			}
		}
		quantity := g(fitem, "quantity")
		if len(iitems) > 0 {
			var iitem *dal.Record
			for _, ii := range iitems {
				if g(rec(g(ii, "inventory")), "id") == g(fitem, "inventory_item_id") {
					iitem = ii
				}
			}
			if iitem != nil || !skipMissing {
				quantity = totals.Div(quantity, g(iitem, "required_quantity"))
			}
		}
		r := dal.NewRecord()
		r.Set("id", lineItemID)
		r.Set("quantity", quantity)
		out = append(out, r)
	}
	return out
}

func findFulfillment(order *dal.Record, id any) *dal.Record {
	for _, f := range recs(g(order, "fulfillments")) {
		if g(f, "id") == id {
			return f
		}
	}
	return nil
}

// CreateOrderShipmentInput mirrors CreateOrderShipmentWorkflowInput.
type CreateOrderShipmentInput struct {
	OrderID       string
	FulfillmentID string
	Input         *dal.Record
}

var createShipmentValidateOrder = composer.CreateStep("create-shipment-validate-order",
	func(ctx composer.StepExecutionContext, in CreateOrderShipmentInput) (composer.StepResponse[any], error) {
		order := in.Input
		if err := throwIfOrderIsCancelled(order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfItemsDoesNotExistsInOrder(order, recs(g(order, "__input_items"))); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if findFulfillment(order, in.FulfillmentID) == nil {
			return composer.StepResponse[any]{}, fmt.Errorf("Fulfillment with id %s not found in the order", in.FulfillmentID)
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateOrderShipmentWorkflow mirrors createOrderShipmentWorkflow.
var CreateOrderShipmentWorkflow = composer.CreateWorkflow("create-order-shipment",
	func(e *composer.Execution, in CreateOrderShipmentInput) (any, error) {
		order, err := queryOrder(e, "get-order", in.OrderID, []string{"id", "status", "region_id", "currency_code", "items.id", "items.quantity",
			"items.variant.manage_inventory", "items.variant.inventory_items.inventory.id", "items.variant.inventory_items.required_quantity",
			"fulfillments.*", "fulfillments.items.id", "fulfillments.items.quantity", "fulfillments.items.line_item_id", "fulfillments.items.inventory_item_id"})
		if err != nil {
			return nil, err
		}
		check := dal.NewRecord()
		for _, k := range order.Keys() {
			v, _ := order.Get(k)
			check.Set(k, v)
		}
		check.Set("__input_items", g(in.Input, "items"))
		if _, err := createShipmentValidateOrder.Run(e, CreateOrderShipmentInput{OrderID: in.OrderID, FulfillmentID: in.FulfillmentID, Input: check}); err != nil {
			return nil, err
		}
		labels := g(in.Input, "labels")
		if labels == nil {
			labels = []any{}
		}
		fdata := dal.NewRecord()
		fdata.Set("id", in.FulfillmentID)
		fdata.Set("labels", labels)
		fdata.Set("marked_shipped_by", g(in.Input, "created_by"))
		fulfillment := findFulfillment(order, in.FulfillmentID)
		shipment := dal.NewRecord()
		shipment.Set("order_id", g(order, "id"))
		shipment.Set("reference", utilsmodulessdk.FULFILLMENT)
		shipment.Set("reference_id", g(fulfillment, "id"))
		shipment.Set("created_by", g(in.Input, "created_by"))
		shipment.Set("items", orderFulfillmentItems(order, fulfillment, false, false))
		var created *dal.Record
		err = composer.Parallelize(
			func() error {
				var err error
				created, err = fulfillmentworkflows.CreateShipmentWorkflow.RunAsStep(e, fdata)
				return err
			},
			func() error { _, err := ordersteps.RegisterOrderShipmentStep.Run(e, shipment); return err },
		)
		if err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("id", g(created, "id"))
		setDefined(ev, "no_notification", in.Input)
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "shipment.created", Data: ev}); err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("shipment", created)
		hook.Set("additional_data", g(in.Input, "additional_data"))
		return nil, composer.CreateHook(e, "shipmentCreated", hook)
	})

// MarkOrderFulfillmentAsDeliveredInput mirrors MarkOrderFulfillmentAsDeliveredWorkflowInput.
type MarkOrderFulfillmentAsDeliveredInput struct {
	OrderID        string
	FulfillmentID  string
	NoNotification any
}

var orderFulfillmentDeliverabilityValidationStep = composer.CreateStep("order-fulfillment-deliverability-validation",
	func(ctx composer.StepExecutionContext, in struct{ Order, Fulfillment *dal.Record }) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if findFulfillment(in.Order, g(in.Fulfillment, "id")) == nil {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Fulfillment with id %v not found in the order", g(in.Fulfillment, "id")))
		}
		return composer.NewStepResponse[any](nil), throwIfItemsDoesNotExistsInOrder(in.Order, recs(g(in.Order, "items")))
	}, nil)

// MarkOrderFulfillmentAsDeliveredWorkflow mirrors markOrderFulfillmentAsDeliveredWorkflow.
var MarkOrderFulfillmentAsDeliveredWorkflow = composer.CreateWorkflow("mark-order-fulfillment-as-delivered-workflow",
	func(e *composer.Execution, in MarkOrderFulfillmentAsDeliveredInput) (any, error) {
		ff := dal.NewRecord()
		ff.Set("id", in.FulfillmentID)
		isList := false
		fres, err := commonsteps.UseQueryGraphStep.Config("get-fulfillment", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "fulfillment", Fields: []string{"id"}, Filters: ff},
			Options:    joiner.Options{ThrowIfKeyNotFound: true},
			IsList:     &isList,
		})
		if err != nil {
			return nil, err
		}
		fulfillment := rec(fres.Data)
		order, err := queryOrder(e, "order-query", in.OrderID, []string{"id", "summary", "total", "currency_code", "region_id", "fulfillments.id",
			"fulfillments.items.id", "fulfillments.items.quantity", "fulfillments.items.line_item_id", "fulfillments.items.inventory_item_id",
			"items.id", "items.quantity", "items.variant.manage_inventory", "items.variant.inventory_items.inventory.id", "items.variant.inventory_items.required_quantity"})
		if err != nil {
			return nil, err
		}
		if _, err := orderFulfillmentDeliverabilityValidationStep.Run(e, struct{ Order, Fulfillment *dal.Record }{order, fulfillment}); err != nil {
			return nil, err
		}
		orderFulfillment := findFulfillment(order, g(fulfillment, "id"))
		delivery := dal.NewRecord()
		delivery.Set("order_id", g(order, "id"))
		delivery.Set("reference", utilsmodulessdk.FULFILLMENT)
		delivery.Set("reference_id", g(orderFulfillment, "id"))
		delivery.Set("items", orderFulfillmentItems(order, orderFulfillment, false, true))
		delivered, err := fulfillmentworkflows.MarkFulfillmentAsDeliveredWorkflow.RunAsStep(e, g(fulfillment, "id"))
		if err != nil {
			return nil, err
		}
		if _, err := ordersteps.RegisterOrderDeliveryStep.Run(e, delivery); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("id", g(delivered, "id"))
		ev.Set("no_notification", in.NoNotification)
		_, err = commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "delivery.created", Data: ev})
		return nil, err
	})

// CancelOrderFulfillmentInput mirrors CancelOrderFulfillmentWorkflowInput.
type CancelOrderFulfillmentInput struct {
	OrderID       string
	FulfillmentID string
	Input         *dal.Record
}

var cancelOrderFulfillmentValidateOrder = composer.CreateStep("cancel-fulfillment-validate-order",
	func(ctx composer.StepExecutionContext, in struct {
		Order         *dal.Record
		FulfillmentID string
	}) (composer.StepResponse[any], error) {
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		f := findFulfillment(in.Order, in.FulfillmentID)
		if f == nil {
			return composer.StepResponse[any]{}, invalidData(fmt.Sprintf("Fulfillment with id %s not found in the order", in.FulfillmentID))
		}
		if g(f, "canceled_at") != nil {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "The fulfillment is already canceled")
		}
		if g(f, "shipped_at") != nil {
			return composer.StepResponse[any]{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "The fulfillment has already been shipped. Shipped fulfillments cannot be canceled")
		}
		var items []*dal.Record
		for _, i := range recs(g(f, "items")) {
			r := dal.NewRecord()
			r.Set("id", g(i, "line_item_id"))
			r.Set("quantity", g(i, "quantity"))
			items = append(items, r)
		}
		return composer.NewStepResponse[any](nil), throwIfItemsDoesNotExistsInOrder(in.Order, items)
	}, nil)

// CancelOrderFulfillmentWorkflow mirrors cancelOrderFulfillmentWorkflow.
var CancelOrderFulfillmentWorkflow = composer.CreateWorkflow("cancel-order-fulfillment",
	func(e *composer.Execution, in CancelOrderFulfillmentInput) (any, error) {
		order, err := queryOrder(e, "get-order", in.OrderID, []string{"id", "status", "items.id", "items.quantity", "items.variant.allow_backorder",
			"items.variant.manage_inventory", "items.variant.inventory_items.inventory.id", "items.variant.inventory_items.required_quantity",
			"fulfillments.id", "fulfillments.canceled_at", "fulfillments.shipped_at", "fulfillments.location_id", "fulfillments.items.id",
			"fulfillments.items.quantity", "fulfillments.items.line_item_id", "fulfillments.items.inventory_item_id"})
		if err != nil {
			return nil, err
		}
		if _, err := cancelOrderFulfillmentValidateOrder.Run(e, struct {
			Order         *dal.Record
			FulfillmentID string
		}{order, in.FulfillmentID}); err != nil {
			return nil, err
		}
		fulfillment := findFulfillment(order, in.FulfillmentID)
		var lineItemIDs []any
		for _, i := range recs(g(fulfillment, "items")) {
			lineItemIDs = append(lineItemIDs, g(i, "line_item_id"))
		}
		reservations, err := reservationsFor(e, lineItemIDs)
		if err != nil {
			return nil, err
		}
		cancelData := dal.NewRecord()
		cancelData.Set("order_id", g(order, "id"))
		cancelData.Set("reference", utilsmodulessdk.FULFILLMENT)
		cancelData.Set("reference_id", g(fulfillment, "id"))
		cancelData.Set("items", orderFulfillmentItems(order, fulfillment, true, false))

		toCreate := []*dal.Record{}
		toUpdate := []*dal.Record{}
		adjustments := []*dal.Record{}
		orderItems := map[any]*dal.Record{}
		for _, i := range recs(g(order, "items")) {
			orderItems[g(i, "id")] = i
		}
		for _, fi := range recs(g(fulfillment, "items")) {
			if !truthy(g(fi, "inventory_item_id")) {
				continue
			}
			orderItem := orderItems[g(fi, "line_item_id")]
			iitem := inventoryItemFor(orderItem, g(fi, "inventory_item_id"))
			if iitem == nil {
				continue
			}
			invID := g(rec(g(iitem, "inventory")), "id")
			var reservation *dal.Record
			for _, r := range reservations {
				if g(r, "inventory_item_id") == invID && g(r, "line_item_id") == g(fi, "line_item_id") {
					reservation = r
					break
				}
			}
			if reservation == nil {
				c := dal.NewRecord()
				c.Set("inventory_item_id", invID)
				c.Set("location_id", g(fulfillment, "location_id"))
				c.Set("quantity", g(fi, "quantity"))
				c.Set("line_item_id", g(fi, "line_item_id"))
				c.Set("allow_backorder", truthy(g(rec(g(orderItem, "variant")), "allow_backorder")))
				toCreate = append(toCreate, c)
			} else {
				u := dal.NewRecord()
				u.Set("id", g(reservation, "id"))
				u.Set("quantity", totals.Add(g(reservation, "quantity"), g(fi, "quantity")))
				toUpdate = append(toUpdate, u)
			}
			a := dal.NewRecord()
			a.Set("inventory_item_id", g(fi, "inventory_item_id"))
			a.Set("location_id", g(fulfillment, "location_id"))
			a.Set("adjustment", g(fi, "quantity"))
			adjustments = append(adjustments, a)
		}
		adjustedLevels, err := inventorysteps.AdjustInventoryLevelsStep.Run(e, adjustments)
		if err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("fulfillment_id", g(fulfillment, "id"))
		setDefined(ev, "no_notification", in.Input)
		var createdReservations []*dal.Record
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "order.fulfillment_canceled", Data: ev})
				return err
			},
			func() error { _, err := ordersteps.CancelOrderFulfillmentStep.Run(e, cancelData); return err },
			func() error {
				var err error
				createdReservations, err = reservationsteps.CreateReservationsStep.Run(e, toCreate)
				return err
			},
			func() error { _, err := reservationsteps.UpdateReservationsStep.Run(e, toUpdate); return err },
		)
		if err != nil {
			return nil, err
		}
		levelEvents, createdEvents, updatedEvents := []any{}, []any{}, []any{}
		for _, l := range adjustedLevels {
			x := dal.NewRecord()
			x.Set("id", g(l, "id"))
			x.Set("order_id", g(order, "id"))
			levelEvents = append(levelEvents, x)
		}
		for _, r := range createdReservations {
			x := dal.NewRecord()
			x.Set("id", g(r, "id"))
			x.Set("order_id", g(order, "id"))
			createdEvents = append(createdEvents, x)
		}
		for _, r := range toUpdate {
			x := dal.NewRecord()
			x.Set("id", g(r, "id"))
			x.Set("order_id", g(order, "id"))
			updatedEvents = append(updatedEvents, x)
		}
		err = composer.Parallelize(
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-inventory-level-updated", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "inventory-level.updated", Data: levelEvents})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-reservation-item-created", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.created", Data: createdEvents})
				return err
			},
			func() error {
				_, err := commonsteps.EmitEventStep.Config("emit-reservation-item-updated", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "reservation-item.updated", Data: updatedEvents})
				return err
			},
		)
		if err != nil {
			return nil, err
		}
		if _, err := fulfillmentworkflows.CancelFulfillmentWorkflow.RunAsStep(e, in.FulfillmentID); err != nil {
			return nil, err
		}
		hook := dal.NewRecord()
		hook.Set("fulfillment", fulfillment)
		hook.Set("additional_data", g(in.Input, "additional_data"))
		return nil, composer.CreateHook(e, "orderFulfillmentCanceled", hook)
	})

var _ = toAnyList
