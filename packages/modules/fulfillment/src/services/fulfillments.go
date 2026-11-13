package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// providerResultData mirrors `data: providerResult.data ?? {}` and
// `labels: providerResult.labels ?? []`.
func providerResultData(res *dal.Record) (any, any) {
	data := get(res, "data")
	if data == nil {
		data = dal.NewRecord()
	}
	labels := get(res, "labels")
	if labels == nil {
		labels = []any{}
	}
	return data, labels
}

func recordOrEmpty(v any) *dal.Record {
	if r, ok := v.(*dal.Record); ok && r != nil {
		return r
	}
	return dal.NewRecord()
}

// CreateFulfillment mirrors createFulfillment.
func (s *FulfillmentModuleService) CreateFulfillment(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	ctx = dal.WithSharedIdentityMap(ctx)
	in := dal.NewRecord()
	for _, k := range data.Keys() {
		if k == "order" || k == "additional_data" {
			continue
		}
		v, _ := data.Get(k)
		in.Set(k, v)
	}
	created, err := s.Internal("Fulfillment").Create(ctx, []*dal.Record{in})
	if err != nil {
		return nil, err
	}
	fulfillment := created[0]
	id := get(fulfillment, "id")
	provider := fmt.Sprint(get(fulfillment, "provider_id"))
	rest := dal.NewRecord()
	for _, k := range fulfillment.Keys() {
		if k == "items" || k == "data" || k == "provider_id" {
			continue
		}
		v, _ := fulfillment.Get(k)
		rest.Set(k, v)
	}
	items, _ := get(fulfillment, "items").([]any)
	if items == nil {
		for _, it := range recordsOf(get(fulfillment, "items")) {
			items = append(items, it)
		}
	}
	res, err := s.provider.CreateFulfillment(ctx, provider, recordOrEmpty(get(fulfillment, "data")), items, recordOrEmpty(get(data, "order")), rest)
	if err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	resData, resLabels := providerResultData(res)
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("data", resData)
	u.Set("labels", resLabels)
	if _, err := s.Internal("Fulfillment").Update(ctx, []any{u}); err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	// the update loads its own instance: the created entity serialized
	// keeps the data and labels it was created with
	return fulfillment, nil
}

// UpdateFulfillment mirrors updateFulfillment.
func (s *FulfillmentModuleService) UpdateFulfillment(ctx context.Context, id any, data *dal.Record) (*dal.Record, error) {
	// one transaction manager: the labels loaded first are the initialized
	// collection the update replaces
	ctx = dal.WithSharedIdentityMap(ctx)
	var out *dal.Record
	err := s.tx(ctx, func(t *FulfillmentModuleService) error {
		existing, err := t.Internal("Fulfillment").Retrieve(ctx, id, utilsmodulessdk.FindConfig{Relations: []string{"items", "labels"}})
		if err != nil {
			return err
		}
		u := dal.NewRecord()
		u.Set("id", id)
		for _, k := range data.Keys() {
			v, _ := data.Get(k)
			if k == "labels" {
				var labels []any
				for _, l := range recordsOf(v) {
					if !l.Has("id") {
						labels = append(labels, l)
						continue
					}
					var current *dal.Record
					for _, el := range recordsOf(get(existing, "labels")) {
						if get(el, "id") == get(l, "id") {
							current = el
						}
					}
					if current == nil || l.Len() == 1 {
						labels = append(labels, l)
						continue
					}
					merged := dal.NewRecord()
					for _, kk := range current.Keys() {
						vv, _ := current.Get(kk)
						merged.Set(kk, vv)
					}
					for _, kk := range l.Keys() {
						vv, _ := l.Get(kk)
						merged.Set(kk, vv)
					}
					labels = append(labels, merged)
				}
				if labels == nil {
					labels = []any{}
				}
				u.Set(k, labels)
				continue
			}
			u.Set(k, v)
		}
		res, err := t.Internal("Fulfillment").Update(ctx, []any{u})
		if err != nil {
			return err
		}
		if list, ok := res.([]*dal.Record); ok && len(list) > 0 {
			out = list[0]
		}
		return nil
	})
	return out, err
}

func recordsOf(v any) []*dal.Record {
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

// CancelFulfillment mirrors cancelFulfillment.
func (s *FulfillmentModuleService) CancelFulfillment(ctx context.Context, id any) (*dal.Record, error) {
	fulfillment, err := s.Internal("Fulfillment").Retrieve(ctx, id, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if get(fulfillment, "shipped_at") != nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Fulfillment with id %v already shipped", get(fulfillment, "id")))
	}
	if get(fulfillment, "delivered_at") != nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Fulfillment with id %v already delivered", get(fulfillment, "id")))
	}
	if get(fulfillment, "canceled_at") != nil {
		return fulfillment, nil
	}
	if _, err := s.provider.CancelFulfillment(ctx, fmt.Sprint(get(fulfillment, "provider_id")), recordOrEmpty(get(fulfillment, "data"))); err != nil {
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("canceled_at", dal.Now())
	res, err := s.Internal("Fulfillment").Update(ctx, u)
	if err != nil {
		return nil, err
	}
	r, _ := res.(*dal.Record)
	return r, nil
}

// CreateReturnFulfillment mirrors createReturnFulfillment (the bundled manual
// provider returns empty data and labels).
func (s *FulfillmentModuleService) CreateReturnFulfillment(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	ctx = dal.WithSharedIdentityMap(ctx)
	in := dal.NewRecord()
	for _, k := range data.Keys() {
		if k == "order" {
			continue
		}
		v, _ := data.Get(k)
		in.Set(k, v)
	}
	created, err := s.Internal("Fulfillment").Create(ctx, []*dal.Record{in})
	if err != nil {
		return nil, err
	}
	fulfillment := created[0]
	id := get(fulfillment, "id")
	shippingOption, err := s.Internal("ShippingOption").Retrieve(ctx, get(fulfillment, "shipping_option_id"), utilsmodulessdk.FindConfig{Select: []string{"id", "name", "data", "metadata"}})
	if err != nil {
		return nil, err
	}
	arg := dal.NewRecord()
	for _, k := range fulfillment.Keys() {
		v, _ := fulfillment.Get(k)
		arg.Set(k, v)
	}
	arg.Set("shipping_option", shippingOption)
	res, err := s.provider.CreateReturn(ctx, fmt.Sprint(get(fulfillment, "provider_id")), arg)
	if err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	resData, resLabels := providerResultData(res)
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("data", resData)
	u.Set("labels", resLabels)
	if _, err := s.Internal("Fulfillment").Update(ctx, []any{u}); err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	// the update loads its own instance: the created entity serialized
	// keeps the data and labels it was created with
	return fulfillment, nil
}

// CalculateShippingOptionsPrices mirrors calculateShippingOptionsPrices: every
// provider's calculatePrice runs (promiseAll rejects with the first error).
func (s *FulfillmentModuleService) CalculateShippingOptionsPrices(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var firstErr error
	out := make([]*dal.Record, 0, len(data))
	for _, d := range data {
		providerID := fmt.Sprint(get(d, "provider_id"))
		price, err := s.provider.CalculatePrice(ctx, providerID, asRecordOrNil(get(d, "optionData")), asRecordOrNil(get(d, "data")), asRecordOrNil(get(d, "context")))
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		out = append(out, price)
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func asRecordOrNil(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

// ValidateFulfillmentData mirrors validateFulfillmentData.
func (s *FulfillmentModuleService) ValidateFulfillmentData(ctx context.Context, providerID string, optionData, data, context_ *dal.Record) (any, error) {
	return s.provider.ValidateFulfillmentData(ctx, providerID, optionData, data, context_)
}

// ValidateFulfillmentOption mirrors validateFulfillmentOption.
func (s *FulfillmentModuleService) ValidateFulfillmentOption(ctx context.Context, providerID string, data *dal.Record) (bool, error) {
	return s.provider.ValidateOption(ctx, providerID, data)
}

// ValidateShippingOptionsForPriceCalculation mirrors
// validateShippingOptionsForPriceCalculation.
func (s *FulfillmentModuleService) ValidateShippingOptionsForPriceCalculation(ctx context.Context, options []*dal.Record) ([]bool, error) {
	var names []string
	for _, o := range options {
		if get(o, "price_type") != "calculated" {
			names = append(names, fmt.Sprint(get(o, "name")))
		}
	}
	if len(names) > 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Cannot calculate price for non-calculated shipping options: "+strings.Join(names, ", "))
	}
	out := make([]bool, 0, len(options))
	var firstErr error
	for _, o := range options {
		ok, err := s.provider.CanCalculate(ctx, fmt.Sprint(get(o, "provider_id")), o)
		if err != nil && firstErr == nil {
			firstErr = err
		}
		out = append(out, ok)
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}
