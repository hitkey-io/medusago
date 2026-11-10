package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// providerCreateFulfillment mirrors the provider's createFulfillment for the
// bundled manual provider.
func (s *FulfillmentModuleService) providerCall(ctx context.Context, providerID string) error {
	if strings.HasPrefix(providerID, "manual_") {
		return nil
	}
	return fmt.Errorf("\nUnable to retrieve the fulfillment provider with id: %s\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.", providerID)
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
	if err := s.providerCall(ctx, provider); err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("data", dal.NewRecord())
	u.Set("labels", []any{})
	if _, err := s.Internal("Fulfillment").Update(ctx, []any{u}); err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	fulfillment.Set("data", dal.NewRecord())
	fulfillment.Set("labels", []any{})
	return fulfillment, nil
}

// UpdateFulfillment mirrors updateFulfillment.
func (s *FulfillmentModuleService) UpdateFulfillment(ctx context.Context, id any, data *dal.Record) (*dal.Record, error) {
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
	if err := s.providerCall(ctx, fmt.Sprint(get(fulfillment, "provider_id"))); err != nil {
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
	if _, err := s.Internal("ShippingOption").Retrieve(ctx, get(fulfillment, "shipping_option_id"), utilsmodulessdk.FindConfig{Select: []string{"id", "name", "data", "metadata"}}); err != nil {
		return nil, err
	}
	if err := s.providerCall(ctx, fmt.Sprint(get(fulfillment, "provider_id"))); err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("data", dal.NewRecord())
	u.Set("labels", []any{})
	if _, err := s.Internal("Fulfillment").Update(ctx, []any{u}); err != nil {
		_, _ = s.Internal("Fulfillment").Delete(ctx, []any{id})
		return nil, err
	}
	fulfillment.Set("data", dal.NewRecord())
	fulfillment.Set("labels", []any{})
	return fulfillment, nil
}
