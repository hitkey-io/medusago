package services

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// CreateReturnItems mirrors createReturnItems.
func (s *OrderModuleService) CreateReturnItems(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	return s.Internal("ReturnItem").Create(ctx, data)
}

// DeleteReturnItems mirrors deleteReturnItems.
func (s *OrderModuleService) DeleteReturnItems(ctx context.Context, ids []any) error {
	_, err := s.Internal("ReturnItem").Delete(ctx, ids)
	return err
}

func (s *OrderModuleService) orderShippingIDs(ctx context.Context, m *utilsmodulessdk.MedusaService, ids []any) ([]any, error) {
	f := dal.NewRecord()
	f.Set("shipping_method_id", ids)
	rel, err := m.Internal("OrderShipping").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
	if err != nil {
		return nil, err
	}
	out := make([]any, len(rel))
	for i, r := range rel {
		out[i] = get(r, "id")
	}
	return out, nil
}

// SoftDeleteOrderShippingMethods mirrors softDeleteOrderShippingMethods: the
// OrderShipping pivots of the methods are soft deleted as well.
func (s *OrderModuleService) SoftDeleteOrderShippingMethods(ctx context.Context, ids []any) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		rel, err := s.orderShippingIDs(ctx, m, ids)
		if err != nil {
			return err
		}
		if _, err := m.Internal("OrderShippingMethod").SoftDelete(ctx, ids); err != nil {
			return err
		}
		_, err = m.Internal("OrderShipping").SoftDelete(ctx, rel)
		return err
	})
}

// RestoreOrderShippingMethods mirrors restoreOrderShippingMethods.
func (s *OrderModuleService) RestoreOrderShippingMethods(ctx context.Context, ids []any) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		f := dal.NewRecord()
		f.Set("shipping_method_id", ids)
		rel, err := m.Internal("OrderShipping").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}, WithDeleted: true})
		if err != nil {
			return err
		}
		var relIDs []any
		for _, r := range rel {
			relIDs = append(relIDs, get(r, "id"))
		}
		if _, err := m.Internal("OrderShippingMethod").Restore(ctx, ids); err != nil {
			return err
		}
		_, err = m.Internal("OrderShipping").Restore(ctx, relIDs)
		return err
	})
}
