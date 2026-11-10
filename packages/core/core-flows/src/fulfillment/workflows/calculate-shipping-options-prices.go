package workflows

import (
	"fmt"

	cartutils "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/utils"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CalculateShippingOptionsPricesInput mirrors CalculateShippingOptionsPricesWorkflowInput.
type CalculateShippingOptionsPricesInput struct {
	CartID          string
	ShippingOptions []*dal.Record // { id, data }
}

func queryGraph(e *composer.Execution, name, entity string, fields []string, filters *dal.Record) ([]*dal.Record, error) {
	res, err := commonsteps.UseQueryGraphStep.Config(name, composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
		GraphInput: query.GraphInput{Entity: entity, Fields: fields, Filters: filters},
	})
	if err != nil {
		return nil, err
	}
	return recs(res.Data), nil
}

func filterBy(key string, v any) *dal.Record {
	f := dal.NewRecord()
	f.Set(key, v)
	return f
}

// CalculateShippingOptionsPricesWorkflow mirrors calculateShippingOptionsPricesWorkflow.
var CalculateShippingOptionsPricesWorkflow = composer.CreateWorkflow("calculate-shipping-options-prices-workflow",
	func(e *composer.Execution, in CalculateShippingOptionsPricesInput) ([]*dal.Record, error) {
		ids := []any{}
		dataOf := map[any]any{}
		for _, so := range in.ShippingOptions {
			id := cartutils.Get(so, "id")
			ids = append(ids, id)
			if _, seen := dataOf[id]; !seen {
				dataOf[id] = cartutils.Get(so, "data")
			}
		}
		options, err := queryGraph(e, "shipping-options-query", "shipping_option",
			[]string{"id", "provider_id", "data", "shipping_profile_id", "service_zone.fulfillment_set_id"}, filterBy("id", ids))
		if err != nil {
			return nil, err
		}
		carts, err := queryGraph(e, "cart-query", "cart", cartutils.CartFieldsForCalculateShippingOptionsPrices, filterBy("id", in.CartID))
		if err != nil {
			return nil, err
		}
		setIDs := []any{}
		for _, so := range options {
			setIDs = append(setIDs, cartutils.Get(cartutils.Rec(so, "service_zone"), "fulfillment_set_id"))
		}
		lfs, err := queryGraph(e, "location-fulfillment-set-query", "location_fulfillment_set",
			[]string{"id", "stock_location_id", "fulfillment_set_id"}, filterBy("fulfillment_set_id", setIDs))
		if err != nil {
			return nil, err
		}
		locationIDs := []any{}
		locationOfSet := map[any]any{}
		for _, l := range lfs {
			locationIDs = append(locationIDs, cartutils.Get(l, "stock_location_id"))
			locationOfSet[cartutils.Get(l, "fulfillment_set_id")] = cartutils.Get(l, "stock_location_id")
		}
		locations, err := queryGraph(e, "location-query", "stock_location", []string{"id", "name", "address.*"}, filterBy("id", locationIDs))
		if err != nil {
			return nil, err
		}
		hookInput := dal.NewRecord()
		hookInput.Set("input", in)
		hookResult, err := composer.CreateHookWithResult(e, "setCalculatedShippingPricingContext", hookInput)
		if err != nil {
			return nil, err
		}
		var cart *dal.Record
		if len(carts) > 0 {
			cart = carts[0]
		}
		data := []*dal.Record{}
		for _, so := range options {
			if cart == nil {
				// cart.items on an undefined cart (TypeError upstream).
				return nil, fmt.Errorf("Cannot read properties of undefined (reading 'items')")
			}
			ctxRec := dal.NewRecord()
			if hr, ok := hookResult.(*dal.Record); ok && hr != nil {
				for _, k := range hr.Keys() {
					ctxRec.Set(k, cartutils.Get(hr, k))
				}
			}
			for _, k := range cart.Keys() {
				ctxRec.Set(k, cartutils.Get(cart, k))
			}
			ctxRec.Set("items", cartutils.FilterCartItemsByShippingProfile(recs(cartutils.Get(cart, "items")), cartutils.Get(so, "shipping_profile_id")))
			var from any
			setLocation := locationOfSet[cartutils.Get(cartutils.Rec(so, "service_zone"), "fulfillment_set_id")]
			for _, l := range locations {
				if cartutils.Get(l, "id") == setLocation {
					from = l
					break
				}
			}
			if from != nil {
				ctxRec.Set("from_location", from)
			}
			soData := dataOf[cartutils.Get(so, "id")]
			if soData == nil {
				soData = dal.NewRecord()
			}
			d := dal.NewRecord()
			d.Set("id", cartutils.Get(so, "id"))
			d.Set("provider_id", cartutils.Get(so, "provider_id"))
			d.Set("optionData", cartutils.Get(so, "data"))
			d.Set("data", soData)
			d.Set("context", ctxRec)
			data = append(data, d)
		}
		return steps.CalculateShippingOptionsPricesStep.Run(e, data)
	})
