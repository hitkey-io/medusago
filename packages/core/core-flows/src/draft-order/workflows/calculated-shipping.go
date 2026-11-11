package workflows

import (
	"fmt"

	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	ordersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/steps"
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// lineItemFieldsForShippingPriceCalculation mirrors
// LINE_ITEM_FIELDS_FOR_SHIPPING_PRICE_CALCULATION.
var lineItemFieldsForShippingPriceCalculation = []string{"id", "quantity", "variant_id", "variant.id", "variant.weight", "variant.length",
	"variant.height", "variant.width", "variant.material", "variant.product.id", "variant.product.shipping_profile.id", "product.id"}

// enrichPreviewLineItemsForShippingPriceCalculation mirrors
// enrichPreviewLineItemsForShippingPriceCalculation.
func enrichPreviewLineItemsForShippingPriceCalculation(previewItems, lineItems []*dal.Record) []*dal.Record {
	byID := map[any]*dal.Record{}
	for _, li := range lineItems {
		byID[g(li, "id")] = li
	}
	out := make([]*dal.Record, 0, len(previewItems))
	for _, item := range previewItems {
		li := byID[g(item, "id")]
		r := dal.NewRecord()
		for _, k := range item.Keys() {
			v, _ := item.Get(k)
			r.Set(k, v)
		}
		r.Set("quantity", g(item, "quantity"))
		variant := g(li, "variant")
		setOrDelete(r, "variant", variant)
		product := g(rec(variant), "product")
		if product == nil {
			product = g(li, "product")
		}
		setOrDelete(r, "product", product)
		out = append(out, r)
	}
	return out
}

// setOrDelete mirrors assigning a value that may be undefined.
func setOrDelete(r *dal.Record, k string, v any) {
	if v == nil {
		r.Delete(k)
		return
	}
	r.Set(k, v)
}

// draftCalculationData builds one CalculateShippingOptionPriceDTO of the
// draft order workflows.
func draftCalculationData(option *dal.Record, orderID, shippingAddress any, items []*dal.Record) *dal.Record {
	c := dal.NewRecord()
	c.Set("id", orderID)
	c.Set("shipping_address", shippingAddress)
	c.Set("items", cartutils.FilterCartItemsByShippingProfile(items, g(option, "shipping_profile_id")))
	if loc := g(rec(g(rec(g(option, "service_zone")), "fulfillment_set")), "location"); loc != nil {
		c.Set("from_location", loc)
	}
	d := dal.NewRecord()
	d.Set("id", g(option, "id"))
	d.Set("optionData", g(option, "data"))
	d.Set("context", c)
	d.Set("provider_id", g(option, "provider_id"))
	return d
}

// previewItems mirrors previewOrderChangeStep + the line items query of the
// draft order workflows: the projected items with their variants.
func previewItems(e *composer.Execution, orderID any, lineItemsQuery string) ([]*dal.Record, error) {
	preview, err := ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(orderID))
	if err != nil {
		return nil, err
	}
	items := recs(g(preview, "items"))
	ids := []any{}
	for _, it := range items {
		ids = append(ids, g(it, "id"))
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	lineItems, err := graphList(e, lineItemsQuery, "order_line_item", f, lineItemFieldsForShippingPriceCalculation)
	if err != nil {
		return nil, err
	}
	return enrichPreviewLineItemsForShippingPriceCalculation(items, lineItems), nil
}

// fetchCalculatedDraftOrderOption mirrors the "option-calculated" branch of
// fetchShippingOptionForDraftOrderWorkflow.
func fetchCalculatedDraftOrderOption(e *composer.Execution, in FetchShippingOptionForDraftOrderInput) (*dal.Record, error) {
	preview, err := ordersteps.PreviewOrderChangeStep.Run(e, fmt.Sprint(in.OrderID))
	if err != nil {
		return nil, err
	}
	of := dal.NewRecord()
	of.Set("id", in.OrderID)
	order, err := graphOne(e, "draft-order-query", "order", of, []string{"id", "shipping_address.*"}, false)
	if err != nil {
		return nil, err
	}
	items := recs(g(preview, "items"))
	ids := []any{}
	for _, it := range items {
		ids = append(ids, g(it, "id"))
	}
	lf := dal.NewRecord()
	lf.Set("id", ids)
	lineItems, err := graphList(e, "draft-order-line-items-query", "order_line_item", lf, lineItemFieldsForShippingPriceCalculation)
	if err != nil {
		return nil, err
	}
	sf := dal.NewRecord()
	sf.Set("id", in.ShippingOptionID)
	option, err := graphOne(e, "calculated-option", "shipping_option", sf, shippingOptionFieldsForPriceCalculation, false)
	if err != nil {
		return nil, err
	}
	enriched := enrichPreviewLineItemsForShippingPriceCalculation(items, lineItems)
	prices, err := fulfillmentsteps.CalculateShippingOptionsPricesStep.Run(e, []*dal.Record{draftCalculationData(option, g(order, "id"), g(order, "shipping_address"), enriched)})
	if err != nil {
		return nil, err
	}
	out := dal.NewRecord()
	out.Set("id", g(option, "id"))
	out.Set("name", g(option, "name"))
	if len(prices) > 0 {
		out.Set("calculated_price", prices[0])
	}
	return out, nil
}

// refreshPendingCalculated mirrors the calculated amounts plan of
// refreshPendingDraftOrderShippingMethodsWorkflow.
func refreshPendingCalculated(e *composer.Execution, orderID any, change *dal.Record, methods, options []*dal.Record) error {
	items, err := previewItems(e, orderID, "refresh-line-items-query")
	if err != nil {
		return err
	}
	order := rec(g(change, "order"))
	optionsByID := map[any]*dal.Record{}
	for _, o := range options {
		optionsByID[g(o, "id")] = o
	}
	actionByMethod := map[any]any{}
	for _, a := range recs(g(change, "actions")) {
		if g(a, "action") == "SHIPPING_ADD" {
			actionByMethod[g(a, "reference_id")] = g(a, "id")
		}
	}
	var data []*dal.Record
	var methodIDs, actionIDs []any
	for _, m := range methods {
		actionID, ok := actionByMethod[g(m, "id")]
		if !ok || actionID == nil {
			continue
		}
		option := optionsByID[g(m, "shipping_option_id")]
		if g(option, "price_type") != "calculated" {
			continue
		}
		data = append(data, draftCalculationData(option, g(order, "id"), g(order, "shipping_address"), items))
		methodIDs = append(methodIDs, g(m, "id"))
		actionIDs = append(actionIDs, actionID)
	}
	if len(data) == 0 {
		return nil
	}
	prices, err := fulfillmentsteps.CalculateShippingOptionsPricesStep.Run(e, data)
	if err != nil {
		return err
	}
	var methodUpdates, actionUpdates []*dal.Record
	for i, id := range methodIDs {
		m := dal.NewRecord()
		m.Set("id", id)
		m.Set("amount", g(prices[i], "calculated_amount"))
		m.Set("is_custom_amount", false)
		methodUpdates = append(methodUpdates, m)
		a := dal.NewRecord()
		a.Set("id", actionIDs[i])
		a.Set("amount", g(prices[i], "calculated_amount"))
		actionUpdates = append(actionUpdates, a)
	}
	if err := composer.Parallelize(
		func() error { _, err := ordersteps.UpdateOrderShippingMethodsStep.Run(e, methodUpdates); return err },
		func() error { _, err := ordersteps.UpdateOrderChangeActionsStep.Run(e, actionUpdates); return err },
	); err != nil {
		return err
	}
	_, err = orderworkflows.UpdateOrderTaxLinesWorkflow.RunAsStep(e, orderworkflows.UpdateOrderTaxLinesInput{OrderID: orderID, ShippingMethodIDs: methodIDs})
	return err
}

// refreshConfirmedCalculated mirrors the calculated amounts plan of
// refreshConfirmedDraftOrderShippingMethodsWorkflow.
func refreshConfirmedCalculated(e *composer.Execution, orderID any, order *dal.Record, options []*dal.Record) error {
	optionsByID := map[any]*dal.Record{}
	for _, o := range options {
		optionsByID[g(o, "id")] = o
	}
	var data []*dal.Record
	var methodIDs []any
	for _, m := range recs(g(order, "shipping_methods")) {
		if truthy(g(m, "is_custom_amount")) {
			continue
		}
		option := optionsByID[g(m, "shipping_option_id")]
		if g(option, "price_type") != "calculated" {
			continue
		}
		data = append(data, draftCalculationData(option, g(order, "id"), g(order, "shipping_address"), recs(g(order, "items"))))
		methodIDs = append(methodIDs, g(m, "id"))
	}
	if len(data) == 0 {
		return nil
	}
	prices, err := fulfillmentsteps.CalculateShippingOptionsPricesStep.Run(e, data)
	if err != nil {
		return err
	}
	var methodUpdates []*dal.Record
	for i, id := range methodIDs {
		m := dal.NewRecord()
		m.Set("id", id)
		m.Set("amount", g(prices[i], "calculated_amount"))
		m.Set("is_custom_amount", false)
		methodUpdates = append(methodUpdates, m)
	}
	if _, err := ordersteps.UpdateOrderShippingMethodsStep.Run(e, methodUpdates); err != nil {
		return err
	}
	_, err = orderworkflows.UpdateOrderTaxLinesWorkflow.RunAsStep(e, orderworkflows.UpdateOrderTaxLinesInput{OrderID: orderID, ShippingMethodIDs: methodIDs})
	return err
}
