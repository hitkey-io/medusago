package services

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/order/src/utils"
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
// It returns what the generated softDeleteOrderShippingMethods returns (the
// linkable keys of the soft deleted entities).
func (s *OrderModuleService) SoftDeleteOrderShippingMethods(ctx context.Context, ids []any) (any, error) {
	var returned any
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		rel, err := s.orderShippingIDs(ctx, m, ids)
		if err != nil {
			return err
		}
		if returned, err = m.Call(ctx, "softDeleteOrderShippingMethods", []any{ids}); err != nil {
			return err
		}
		_, err = m.Internal("OrderShipping").SoftDelete(ctx, rel)
		return err
	})
	return returned, err
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

// CancelReturn mirrors cancelReturn (services/actions/cancel-return.ts): an
// order change cancelling the return items and removing its shipping
// methods is created and confirmed, and the return is marked canceled.
func (s *OrderModuleService) CancelReturn(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	var out *dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		ret, err := t.RetrieveRelated(ctx, "Return", fmt.Sprint(get(data, "return_id")), utilsmodulessdk.FindConfig{
			Select:    []string{"id", "order_id", "items.item_id", "items.quantity", "items.received_quantity"},
			Relations: []string{"items", "shipping_methods"},
		})
		if err != nil {
			return err
		}
		out = ret
		var actions []any
		for _, item := range records(get(ret, "items")) {
			a := dal.NewRecord()
			a.Set("action", utils.CancelReturnItem)
			a.Set("order_id", get(ret, "order_id"))
			a.Set("return_id", get(ret, "id"))
			a.Set("reference", "return")
			a.Set("reference_id", get(ret, "id"))
			d := dal.NewRecord()
			d.Set("reference_id", get(item, "item_id"))
			d.Set("quantity", get(item, "quantity"))
			a.Set("details", d)
			actions = append(actions, a)
		}
		for _, sm := range records(get(ret, "shipping_methods")) {
			a := dal.NewRecord()
			a.Set("action", utils.ShippingRemove)
			a.Set("order_id", get(ret, "order_id"))
			a.Set("return_id", get(ret, "id"))
			a.Set("reference", "return")
			a.Set("reference_id", get(sm, "id"))
			amount := get(sm, "raw_amount")
			if amount == nil {
				amount = get(sm, "amount")
			}
			a.Set("amount", amount)
			actions = append(actions, a)
		}
		c := dal.NewRecord()
		c.Set("order_id", get(ret, "order_id"))
		c.Set("return_id", get(ret, "id"))
		c.Set("reference", "return")
		c.Set("reference_id", get(ret, "id"))
		for _, k := range []string{"description", "internal_note", "created_by", "metadata"} {
			if v, ok := data.Get(k); ok {
				c.Set(k, v)
			}
		}
		if actions == nil {
			actions = []any{}
		}
		c.Set("actions", actions)
		changes, err := t.createOrderChange(ctx, []*dal.Record{c})
		if err != nil {
			return err
		}
		// the order was referenced by the loaded entity before
		// createOrderChange_ loaded it: the new change does not mark it
		// changed
		dal.Untouch(t.DB(), "Order")
		u := dal.NewRecord()
		u.Set("canceled_at", dal.Now())
		u.Set("status", "canceled")
		sel := dal.NewRecord()
		sel.Set("id", get(ret, "id"))
		upd := dal.NewRecord()
		upd.Set("selector", sel)
		upd.Set("data", u)
		if _, err := m.Internal("Return").Update(ctx, []any{upd}); err != nil {
			return err
		}
		_, err = t.ConfirmOrderChange(ctx, []any{fmt.Sprint(get(changes[0], "id"))})
		return err
	})
	return out, err
}

// cancelRelated creates and confirms the order change cancelling a claim or
// an exchange and marks it canceled (services/actions/cancel-claim.ts and
// cancel-exchange.ts).
func (s *OrderModuleService) cancelRelated(ctx context.Context, data *dal.Record, className, idKey, key, reference string, sel []string, rels []string, build func(entity *dal.Record) []any) (*dal.Record, error) {
	var out *dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		entity, err := t.RetrieveRelated(ctx, className, fmt.Sprint(get(data, idKey)), utilsmodulessdk.FindConfig{Select: sel, Relations: rels})
		if err != nil {
			return err
		}
		out = entity
		actions := build(entity)
		for _, sm := range records(get(entity, "shipping_methods")) {
			a := dal.NewRecord()
			a.Set("action", utils.ShippingRemove)
			a.Set("order_id", get(entity, "order_id"))
			a.Set(key, get(entity, "id"))
			a.Set("reference", reference)
			a.Set("reference_id", get(sm, "id"))
			amount := get(sm, "raw_amount")
			if amount == nil {
				amount = get(sm, "amount")
			}
			a.Set("amount", amount)
			actions = append(actions, a)
		}
		c := dal.NewRecord()
		c.Set("order_id", get(entity, "order_id"))
		c.Set(key, get(entity, "id"))
		c.Set("reference", reference)
		c.Set("reference_id", get(entity, "id"))
		for _, k := range []string{"description", "internal_note", "created_by", "metadata"} {
			if v, ok := data.Get(k); ok {
				c.Set(k, v)
			}
		}
		if actions == nil {
			actions = []any{}
		}
		c.Set("actions", actions)
		changes, err := t.createOrderChange(ctx, []*dal.Record{c})
		if err != nil {
			return err
		}
		// the order was referenced by the loaded entity before
		// createOrderChange_ loaded it: the new change does not mark it
		// changed
		dal.Untouch(t.DB(), "Order")
		u := dal.NewRecord()
		u.Set("canceled_at", dal.Now())
		selector := dal.NewRecord()
		selector.Set("id", get(entity, "id"))
		upd := dal.NewRecord()
		upd.Set("selector", selector)
		upd.Set("data", u)
		if _, err := m.Internal(className).Update(ctx, []any{upd}); err != nil {
			return err
		}
		_, err = t.ConfirmOrderChange(ctx, []any{fmt.Sprint(get(changes[0], "id"))})
		return err
	})
	return out, err
}

func itemAction(action string, entity *dal.Record, key, reference string, withOrder bool, detail func(d *dal.Record)) *dal.Record {
	a := dal.NewRecord()
	a.Set("action", action)
	if withOrder {
		a.Set("order_id", get(entity, "order_id"))
	}
	a.Set(key, get(entity, "id"))
	a.Set("reference", reference)
	a.Set("reference_id", get(entity, "id"))
	d := dal.NewRecord()
	detail(d)
	a.Set("details", d)
	return a
}

// CancelClaim mirrors cancelClaim.
func (s *OrderModuleService) CancelClaim(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	return s.cancelRelated(ctx, data, "OrderClaim", "claim_id", "claim_id", "claim",
		[]string{"id", "order_id", "claim_items.item_id", "claim_items.is_additional_item", "claim_items.quantity", "additional_items.id",
			"additional_items.item_id", "additional_items.quantity", "additional_items.is_additional_item"},
		[]string{"claim_items", "additional_items", "shipping_methods"},
		func(claim *dal.Record) []any {
			var actions []any
			for _, item := range records(get(claim, "claim_items")) {
				actions = append(actions, itemAction(utils.ReinstateItem, claim, "claim_id", "claim", false, func(d *dal.Record) {
					d.Set("reference_id", get(item, "item_id"))
					d.Set("quantity", get(item, "quantity"))
				}))
			}
			for _, item := range records(get(claim, "additional_items")) {
				actions = append(actions, itemAction(utils.ItemRemove, claim, "claim_id", "claim", true, func(d *dal.Record) {
					d.Set("reference_id", get(item, "item_id"))
					d.Set("quantity", get(item, "quantity"))
				}))
			}
			return actions
		})
}

// CancelExchange mirrors cancelExchange.
func (s *OrderModuleService) CancelExchange(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	return s.cancelRelated(ctx, data, "OrderExchange", "exchange_id", "exchange_id", "exchange",
		[]string{"id", "order_id", "additional_items.id", "additional_items.item_id", "additional_items.quantity"},
		[]string{"additional_items", "shipping_methods"},
		func(exchange *dal.Record) []any {
			var actions []any
			for _, item := range records(get(exchange, "additional_items")) {
				actions = append(actions, itemAction(utils.ItemRemove, exchange, "exchange_id", "exchange", true, func(d *dal.Record) {
					d.Set("order_id", get(exchange, "order_id"))
					d.Set("reference_id", get(item, "item_id"))
					d.Set("exchange_id", get(exchange, "id"))
					d.Set("quantity", get(item, "quantity"))
				}))
			}
			return actions
		})
}
