package workflows

import (
	"fmt"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	fulfillmentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// prepareCompleteReturnShippingMethod mirrors prepareShippingMethodData.
func prepareCompleteReturnShippingMethod(orderID any, inputShipping, option *dal.Record) (*dal.Record, error) {
	if inputShipping == nil {
		return nil, nil
	}
	if option == nil {
		return nil, fmt.Errorf("Cannot read properties of undefined (reading 'name')")
	}
	obj := dal.NewRecord()
	obj.Set("name", g(option, "name"))
	obj.Set("order_id", orderID)
	obj.Set("shipping_option_id", g(option, "id"))
	obj.Set("amount", 0)
	obj.Set("is_tax_inclusive", false)
	obj.Set("data", dal.NewRecord())
	obj.Set("tax_lines", []any{})
	obj.Set("adjustments", []any{})
	if price := g(inputShipping, "price"); price != nil && totals.Gte(price, 0) {
		obj.Set("amount", price)
	} else if g(option, "price_type") != "calculated" {
		cp := rec(g(option, "calculated_price"))
		if cp == nil {
			return nil, fmt.Errorf("Cannot read properties of undefined (reading 'calculated_amount')")
		}
		obj.Set("amount", g(cp, "calculated_amount"))
		obj.Set("is_tax_inclusive", truthy(g(cp, "is_calculated_price_tax_inclusive")))
	}
	return obj, nil
}

// prepareCompleteReturnFulfillmentData mirrors prepareFulfillmentData.
func prepareCompleteReturnFulfillmentData(order, input, option *dal.Record) (*dal.Record, error) {
	orderItems := map[any]*dal.Record{}
	for _, i := range recs(g(order, "items")) {
		orderItems[g(i, "id")] = i
	}
	items := []any{}
	for _, i := range recs(g(input, "items")) {
		oi := orderItems[g(i, "id")]
		fi := dal.NewRecord()
		fi.Set("line_item_id", g(i, "id"))
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
		items = append(items, fi)
	}
	location := rec(g(rec(g(rec(g(option, "service_zone")), "fulfillment_set")), "location"))
	locationID := g(input, "location_id")
	if !truthy(locationID) {
		locationID = g(location, "id")
	}
	if !truthy(locationID) {
		return nil, invalidData(fmt.Sprintf("Cannot create return without stock location, either provide a location or you should link the shipping option %v to a stock location.", g(option, "id")))
	}
	addr := dal.NewRecord()
	if address := rec(g(location, "address")); address != nil {
		for _, k := range address.Keys() {
			if k == "id" {
				continue
			}
			v, _ := address.Get(k)
			addr.Set(k, v)
		}
	}
	shipping := rec(g(input, "return_shipping"))
	labels := g(shipping, "labels")
	if labels == nil {
		labels = []any{}
	}
	in := dal.NewRecord()
	in.Set("location_id", locationID)
	in.Set("provider_id", g(option, "provider_id"))
	in.Set("shipping_option_id", g(shipping, "option_id"))
	in.Set("items", items)
	in.Set("labels", labels)
	in.Set("delivery_address", addr)
	in.Set("order", order)
	return in, nil
}

// CreateCompleteReturnValidationStep mirrors createCompleteReturnValidationStep.
var CreateCompleteReturnValidationStep = composer.CreateStep("create-complete-return-validation",
	func(ctx composer.StepExecutionContext, in struct{ Order, Input *dal.Record }) (composer.StepResponse[any], error) {
		items, ok := in.Input.Get("items")
		if !ok || items == nil {
			return composer.StepResponse[any]{}, invalidData("Items are required to create a return.")
		}
		if err := throwIfOrderIsCancelled(in.Order); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := throwIfItemsDoesNotExistsInOrder(in.Order, recs(items)); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if err := validateReturnReasons(ctx, g(in.Input, "order_id"), recs(items)); err != nil {
			return composer.StepResponse[any]{}, err
		}
		if refund := g(in.Input, "refund_amount"); truthy(refund) && totals.Gt(refund, g(in.Order, "item_total")) {
			return composer.StepResponse[any]{}, invalidData("Refund amount cannot be greater than order total.")
		}
		return composer.NewStepResponse[any](nil), nil
	}, nil)

// CreateAndCompleteReturnOrderWorkflow mirrors createAndCompleteReturnOrderWorkflow.
var CreateAndCompleteReturnOrderWorkflow = composer.CreateWorkflow("create-complete-return-order",
	func(e *composer.Execution, input *dal.Record) (*dal.Record, error) {
		order, err := remoteOne(e, "", "orders", idVars(g(input, "order_id")),
			[]string{"id", "status", "region_id", "currency_code", "total", "item_total", "items.*"}, true)
		if err != nil {
			return nil, err
		}
		if _, err := CreateCompleteReturnValidationStep.Run(e, struct{ Order, Input *dal.Record }{order, input}); err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("order", order)
		hookInput.Set("additional_data", g(input, "additional_data"))
		pricingContext, err := composer.CreateHookWithResult(e, "setPricingContext", hookInput)
		if err != nil {
			return nil, err
		}
		shipping := rec(g(input, "return_shipping"))
		ctxRec := dal.NewRecord()
		if pc, ok := pricingContext.(*dal.Record); ok && pc != nil {
			for _, k := range pc.Keys() {
				v, _ := pc.Get(k)
				ctxRec.Set(k, v)
			}
		}
		ctxRec.Set("currency_code", g(order, "currency_code"))
		if truthy(g(order, "region_id")) {
			ctxRec.Set("region_id", g(order, "region_id"))
		}
		cp := dal.NewRecord()
		cp.Set("context", ctxRec)
		vars := dal.NewRecord()
		vars.Set("id", g(shipping, "option_id"))
		vars.Set("calculated_price", cp)
		option, err := remoteOne(e, "return-shipping-option", "shipping_options", vars, []string{
			"id", "price_type", "name", "provider_id", "calculated_price.calculated_amount", "calculated_price.is_calculated_price_tax_inclusive",
			"service_zone.fulfillment_set.location.id", "service_zone.fulfillment_set.location.address.*",
		}, false)
		if err != nil {
			return nil, err
		}
		shippingMethod, err := prepareCompleteReturnShippingMethod(g(input, "order_id"), shipping, option)
		if err != nil {
			return nil, err
		}
		fulfillmentData, err := prepareCompleteReturnFulfillmentData(order, input, option)
		if err != nil {
			return nil, err
		}
		fulfillment, err := fulfillmentworkflows.CreateReturnFulfillmentWorkflow.RunAsStep(e, fulfillmentData)
		if err != nil {
			return nil, err
		}
		data := dal.NewRecord()
		data.Set("order_id", g(input, "order_id"))
		setIfPresent(data, input, "location_id", "items")
		if shippingMethod != nil {
			data.Set("shipping_method", shippingMethod)
		}
		setIfPresent(data, input, "refund_amount", "created_by")
		created, err := ordersteps.CreateCompleteReturnStep.Run(e, data)
		if err != nil {
			return nil, err
		}
		o := dal.NewRecord()
		o.Set("return_id", g(created, "id"))
		f := dal.NewRecord()
		f.Set("fulfillment_id", g(fulfillment, "id"))
		link := dal.NewRecord()
		link.Set(utilsmodulessdk.ORDER, o)
		link.Set(utilsmodulessdk.FULFILLMENT, f)
		if _, err := commonsteps.CreateRemoteLinkStep.Run(e, []*dal.Record{link}); err != nil {
			return nil, err
		}
		receive := dal.NewRecord()
		receive.Set("return_id", g(created, "id"))
		receiveNow := g(input, "receive_now") == true
		if receiveNow {
			items := []any{}
			for _, i := range recs(g(order, "items")) {
				r := dal.NewRecord()
				r.Set("id", g(i, "id"))
				r.Set("quantity", g(i, "quantity"))
				items = append(items, r)
			}
			receive.Set("items", items)
			receive.Set("created_by", g(input, "created_by"))
		} else {
			receive.Set("items", []any{})
		}
		if _, err := ordersteps.ReceiveReturnStep.Run(e, receive); err != nil {
			return nil, err
		}
		ev := dal.NewRecord()
		ev.Set("order_id", g(order, "id"))
		ev.Set("return_id", g(created, "id"))
		if _, err := commonsteps.EmitEventStep.Config("emit-return-requested-event", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "order.return_requested", Data: ev}); err != nil {
			return nil, err
		}
		if receiveNow {
			if _, err := commonsteps.EmitEventStep.Config("emit-return-received-event", composer.StepOptions{}).Run(e, commonsteps.EmitEventStepInput{EventName: "order.return_received", Data: ev}); err != nil {
				return nil, err
			}
		}
		return created, nil
	})
