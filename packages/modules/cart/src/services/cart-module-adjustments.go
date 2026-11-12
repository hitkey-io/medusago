package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// The cart service methods writing shipping methods, adjustments and tax
// lines (cart-module.ts). Upstream overloads take either a cart id with the
// rows of that cart or the rows alone (bulk): cartID "" is the bulk form.

func (s *CartModuleService) cartRelationIDs(ctx context.Context, cartID, relation string) (map[any]bool, error) {
	cart, err := s.Internal("Cart").Retrieve(ctx, cartID, utilsmodulessdk.FindConfig{Select: []string{"id"}, Relations: []string{relation}})
	if err != nil {
		return nil, err
	}
	ids := map[any]bool{}
	list, _ := get(cart, relation).([]any)
	for _, r := range list {
		if rec, ok := r.(*dal.Record); ok {
			ids[get(rec, "id")] = true
		}
	}
	if rs, ok := get(cart, relation).([]*dal.Record); ok {
		for _, rec := range rs {
			ids[get(rec, "id")] = true
		}
	}
	return ids, nil
}

// AddShippingMethods mirrors addShippingMethods: with a cart id the methods
// get its cart_id (the cart must exist).
func (s *CartModuleService) AddShippingMethods(ctx context.Context, cartID string, data []*dal.Record) ([]*dal.Record, error) {
	if cartID != "" {
		cart, err := s.Internal("Cart").Retrieve(ctx, cartID, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return nil, err
		}
		for _, d := range data {
			d.Set("cart_id", get(cart, "id"))
		}
	}
	return s.Internal("ShippingMethod").Create(ctx, data)
}

// AddLineItemAdjustments mirrors addLineItemAdjustments: with a cart id every
// adjustment's item must be a line item of the cart.
func (s *CartModuleService) AddLineItemAdjustments(ctx context.Context, cartID string, adjustments []*dal.Record) ([]*dal.Record, error) {
	if cartID != "" {
		ids, err := s.cartRelationIDs(ctx, cartID, "items")
		if err != nil {
			return nil, err
		}
		for _, a := range adjustments {
			if !ids[get(a, "item_id")] {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Line item with id %v does not exist on cart with id %s", jsString(get(a, "item_id")), cartID))
			}
		}
	}
	return s.Internal("LineItemAdjustment").Create(ctx, adjustments)
}

// AddShippingMethodAdjustments mirrors addShippingMethodAdjustments: with a
// cart id every adjustment's shipping method must be one of the cart's.
func (s *CartModuleService) AddShippingMethodAdjustments(ctx context.Context, cartID string, adjustments []*dal.Record) ([]*dal.Record, error) {
	if cartID != "" {
		ids, err := s.cartRelationIDs(ctx, cartID, "shipping_methods")
		if err != nil {
			return nil, err
		}
		for _, a := range adjustments {
			if !ids[get(a, "shipping_method_id")] {
				return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Shipping method with id %v does not exist on cart with id %s", jsString(get(a, "shipping_method_id")), cartID))
			}
		}
	}
	return s.Internal("ShippingMethodAdjustment").Create(ctx, adjustments)
}

// setAdjustments mirrors setLineItemAdjustments_ / setShippingMethodAdjustments_:
// the cart's adjustments missing from the input (by id) are soft deleted,
// then the input is upserted.
func (s *CartModuleService) setAdjustments(ctx context.Context, cartID, className, relation string, adjustments []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		cart, err := m.Internal("Cart").Retrieve(ctx, cartID, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		rel := dal.NewRecord()
		rel.Set("cart_id", get(cart, "id"))
		f := dal.NewRecord()
		f.Set(relation, rel)
		existing, err := m.Internal(className).List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
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
		for _, e := range existing {
			if !keep[get(e, "id")] {
				toDelete = append(toDelete, get(e, "id"))
			}
		}
		if len(toDelete) > 0 {
			if _, err := m.Internal(className).SoftDelete(ctx, toDelete); err != nil {
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

// SetLineItemAdjustments mirrors setLineItemAdjustments.
func (s *CartModuleService) SetLineItemAdjustments(ctx context.Context, cartID string, adjustments []*dal.Record) ([]*dal.Record, error) {
	return s.setAdjustments(ctx, cartID, "LineItemAdjustment", "item", adjustments)
}

// SetShippingMethodAdjustments mirrors setShippingMethodAdjustments.
func (s *CartModuleService) SetShippingMethodAdjustments(ctx context.Context, cartID string, adjustments []*dal.Record) ([]*dal.Record, error) {
	return s.setAdjustments(ctx, cartID, "ShippingMethodAdjustment", "shipping_method", adjustments)
}

// AddLineItemTaxLines mirrors addLineItemTaxLines: with a cart id the cart
// must exist.
func (s *CartModuleService) AddLineItemTaxLines(ctx context.Context, cartID string, taxLines []*dal.Record) ([]*dal.Record, error) {
	if cartID != "" {
		if _, err := s.Internal("Cart").Retrieve(ctx, cartID, utilsmodulessdk.FindConfig{Select: []string{"id"}}); err != nil {
			return nil, err
		}
	}
	return s.Internal("LineItemTaxLine").Create(ctx, taxLines)
}

// AddShippingMethodTaxLines mirrors addShippingMethodTaxLines.
func (s *CartModuleService) AddShippingMethodTaxLines(ctx context.Context, cartID string, taxLines []*dal.Record) ([]*dal.Record, error) {
	if cartID != "" {
		if _, err := s.Internal("Cart").Retrieve(ctx, cartID, utilsmodulessdk.FindConfig{Select: []string{"id"}}); err != nil {
			return nil, err
		}
	}
	return s.Internal("ShippingMethodTaxLine").Create(ctx, taxLines)
}

// jsString renders a value like a JS template literal (undefined/null).
func jsString(v any) string {
	if v == nil {
		return "undefined"
	}
	return fmt.Sprint(v)
}
