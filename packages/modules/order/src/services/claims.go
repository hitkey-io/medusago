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
