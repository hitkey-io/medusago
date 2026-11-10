package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// CreateOrderLineItems mirrors createOrderLineItems(data[]): items with an
// order_id get the order version and an OrderItem row.
func (s *OrderModuleService) CreateOrderLineItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		var orderIDs []any
		for _, d := range data {
			orderIDs = append(orderIDs, get(d, "order_id"))
		}
		f := dal.NewRecord()
		f.Set("id", orderIDs)
		orders, err := m.Internal("Order").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
		if err != nil {
			return err
		}
		versions := map[string]any{}
		for _, o := range orders {
			versions[fmt.Sprint(get(o, "id"))] = get(o, "version")
		}
		in := make([]*dal.Record, len(data))
		for i, d := range data {
			in[i] = copyRecord(d)
			if get(d, "order_id") != nil {
				in[i].Set("version", versions[fmt.Sprint(get(d, "order_id"))])
			}
		}
		out, err = t.createOrderLineItemsBulk(ctx, in)
		return err
	})
	return out, err
}

// DeleteOrderLineItems mirrors deleteOrderLineItems.
func (s *OrderModuleService) DeleteOrderLineItems(ctx context.Context, ids []any) error {
	_, err := s.Internal("OrderLineItem").Delete(ctx, ids)
	return err
}

// CreateOrderClaimItems mirrors createOrderClaimItems.
func (s *OrderModuleService) CreateOrderClaimItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("OrderClaimItem").Create(ctx, data)
}

// DeleteOrderClaimItems mirrors deleteOrderClaimItems.
func (s *OrderModuleService) DeleteOrderClaimItems(ctx context.Context, ids []any) error {
	_, err := s.Internal("OrderClaimItem").Delete(ctx, ids)
	return err
}

// CreateOrderExchangeItems mirrors createOrderExchangeItems.
func (s *OrderModuleService) CreateOrderExchangeItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("OrderExchangeItem").Create(ctx, data)
}

// DeleteOrderExchangeItems mirrors deleteOrderExchangeItems.
func (s *OrderModuleService) DeleteOrderExchangeItems(ctx context.Context, ids []any) error {
	_, err := s.Internal("OrderExchangeItem").Delete(ctx, ids)
	return err
}

// UpdateOrdersReturning mirrors updateOrders(data[]) and returns the updated
// orders.
func (s *OrderModuleService) UpdateOrdersReturning(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	res, err := s.Internal("Order").Update(ctx, toAny(data))
	if err != nil {
		return nil, err
	}
	if r, ok := res.(*dal.Record); ok {
		return []*dal.Record{r}, nil
	}
	out, _ := res.([]*dal.Record)
	return out, nil
}

// DeleteOrders mirrors deleteOrders: addresses, order changes, line items and
// shipping methods of the orders are hard deleted with them.
func (s *OrderModuleService) DeleteOrders(ctx context.Context, ids []any) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		f := dal.NewRecord()
		f.Set("id", ids)
		orders, err := m.Internal("Order").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "shipping_address_id", "billing_address_id"}})
		if err != nil {
			return err
		}
		var addressIDs []any
		for _, o := range orders {
			for _, k := range []string{"shipping_address_id", "billing_address_id"} {
				if v := get(o, k); v != nil && v != "" {
					addressIDs = append(addressIDs, v)
				}
			}
		}
		byOrder := dal.NewRecord()
		byOrder.Set("order_id", ids)
		changes, err := m.Internal("OrderChange").List(ctx, byOrder, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		items, err := m.Internal("OrderItem").List(ctx, byOrder, utilsmodulessdk.FindConfig{Select: []string{"id", "item_id"}})
		if err != nil {
			return err
		}
		shipping, err := m.Internal("OrderShipping").List(ctx, byOrder, utilsmodulessdk.FindConfig{Select: []string{"shipping_method_id"}})
		if err != nil {
			return err
		}
		if len(addressIDs) > 0 {
			if _, err := m.Internal("OrderAddress").Delete(ctx, addressIDs); err != nil {
				return err
			}
		}
		if len(changes) > 0 {
			var changeIDs []any
			for _, c := range changes {
				changeIDs = append(changeIDs, get(c, "id"))
			}
			if _, err := m.Internal("OrderChange").Delete(ctx, changeIDs); err != nil {
				return err
			}
		}
		if _, err := m.Internal("Order").Delete(ctx, ids); err != nil {
			return err
		}
		var lineItemIDs, methodIDs []any
		for _, it := range items {
			lineItemIDs = append(lineItemIDs, get(it, "item_id"))
		}
		for _, sm := range shipping {
			methodIDs = append(methodIDs, get(sm, "shipping_method_id"))
		}
		if len(lineItemIDs) > 0 {
			if _, err := m.Internal("OrderLineItem").Delete(ctx, lineItemIDs); err != nil {
				return err
			}
		}
		if len(methodIDs) > 0 {
			if _, err := m.Internal("OrderShippingMethod").Delete(ctx, methodIDs); err != nil {
				return err
			}
		}
		return nil
	})
}
