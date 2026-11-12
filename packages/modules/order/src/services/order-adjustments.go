package services

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

func adjustmentIDsOf(rows []*dal.Record, nested string) []any {
	var ids []any
	for _, r := range rows {
		adj := records(get(r, "adjustments"))
		if len(adj) == 0 && nested != "" {
			adj = records(get(rec(get(r, nested)), "adjustments"))
		}
		for _, a := range adj {
			ids = append(ids, get(a, "id"))
		}
	}
	return ids
}

// setOrderAdjustments mirrors setOrderLineItemAdjustments_ /
// setOrderShippingMethodAdjustments_: the order's adjustments missing from
// the input (by id) are deleted, then the input is upserted.
func (s *OrderModuleService) setOrderAdjustments(ctx context.Context, orderID, className, relation, nested string, adjustments []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		order, err := s.tx(m).RetrieveOrder(ctx, orderID, utilsmodulessdk.FindConfig{Select: []string{"id"}, Relations: []string{relation}})
		if err != nil {
			return err
		}
		keep := map[any]bool{}
		for _, a := range adjustments {
			if id := get(a, "id"); id != nil && id != "" {
				keep[id] = true
			}
		}
		var toDelete []any
		rootRelation := relation
		for i, c := range relation {
			if c == '.' {
				rootRelation = relation[:i]
				break
			}
		}
		for _, id := range adjustmentIDsOf(records(get(order, rootRelation)), nested) {
			if !keep[id] {
				toDelete = append(toDelete, id)
			}
		}
		if len(toDelete) > 0 {
			if _, err := m.Internal(className).Delete(ctx, toDelete); err != nil {
				return err
			}
		}
		out, err = m.Internal(className).Upsert(ctx, adjustments)
		return err
	})
	if out == nil {
		out = []*dal.Record{}
	}
	return out, err
}

// SetOrderLineItemAdjustments mirrors setOrderLineItemAdjustments.
func (s *OrderModuleService) SetOrderLineItemAdjustments(ctx context.Context, orderID string, adjustments []*dal.Record) ([]*dal.Record, error) {
	return s.setOrderAdjustments(ctx, orderID, "OrderLineItemAdjustment", "items.item.adjustments", "item", adjustments)
}

// SetOrderShippingMethodAdjustments mirrors setOrderShippingMethodAdjustments.
func (s *OrderModuleService) SetOrderShippingMethodAdjustments(ctx context.Context, orderID string, adjustments []*dal.Record) ([]*dal.Record, error) {
	return s.setOrderAdjustments(ctx, orderID, "OrderShippingMethodAdjustment", "shipping_methods.adjustments", "shipping_method", adjustments)
}
