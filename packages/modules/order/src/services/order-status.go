package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

func (s *OrderModuleService) setStatus(ctx context.Context, ids []any, check func(status any) bool, errMsg string, status string, extra func(u *dal.Record)) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		f := dal.NewRecord()
		f.Set("id", ids)
		orders, _, err := t.listOrders(dal.WithSharedIdentityMap(ctx), f, utilsmodulessdk.FindConfig{}, false)
		if err != nil {
			return err
		}
		var notAllowed []string
		for _, o := range orders {
			if check != nil && !check(get(o, "status")) {
				notAllowed = append(notAllowed, fmt.Sprint(get(o, "id")))
			}
			o.Set("status", status)
		}
		if len(notAllowed) > 0 {
			return invalidData(fmt.Sprintf(errMsg, strings.Join(notAllowed, ", ")))
		}
		var updates []any
		for _, id := range ids {
			u := dal.NewRecord()
			u.Set("id", id)
			u.Set("status", status)
			if extra != nil {
				extra(u)
			}
			updates = append(updates, u)
		}
		if _, err := m.Internal("Order").Update(ctx, updates); err != nil {
			return err
		}
		if extra != nil {
			for _, o := range orders {
				extra(o)
			}
		}
		out = orders
		return nil
	})
	return out, err
}

// Archive mirrors archive.
func (s *OrderModuleService) Archive(ctx context.Context, ids []any) ([]*dal.Record, error) {
	return s.setStatus(ctx, ids, func(st any) bool { return st == "completed" || st == "canceled" || st == "draft" },
		"Orders %s are completed, canceled, or in draft and cannot be archived", "archived", nil)
}

// CompleteOrder mirrors completeOrder.
func (s *OrderModuleService) CompleteOrder(ctx context.Context, ids []any) ([]*dal.Record, error) {
	return s.setStatus(ctx, ids, func(st any) bool { return st != "canceled" }, "Orders %s are canceled and cannot be completed", "completed", nil)
}

// Cancel mirrors cancel.
func (s *OrderModuleService) Cancel(ctx context.Context, ids []any) ([]*dal.Record, error) {
	now := dal.Now()
	return s.setStatus(ctx, ids, nil, "", "canceled", func(u *dal.Record) { u.Set("canceled_at", now) })
}

// AddOrderAction mirrors addOrderAction (array input).
func (s *OrderModuleService) AddOrderAction(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		in := make([]*dal.Record, len(data))
		byChange := map[string][]*dal.Record{}
		var changeIDs []any
		for i, d := range data {
			in[i] = copyRecord(d)
			if id := get(d, "order_change_id"); id != nil {
				k := fmt.Sprint(id)
				byChange[k] = append(byChange[k], in[i])
				changeIDs = append(changeIDs, id)
			}
		}
		if len(changeIDs) > 0 {
			changes, err := t.getAndValidateOrderChange(ctx, changeIDs, false)
			if err != nil {
				return err
			}
			for _, c := range changes {
				for _, a := range byChange[fmt.Sprint(get(c, "id"))] {
					a.Set("order_id", get(c, "order_id"))
					a.Set("version", get(c, "version"))
				}
			}
		}
		var err error
		out, err = m.Internal("OrderChangeAction").Create(ctx, in)
		return err
	})
	return out, err
}

// UpdateOrders mirrors updateOrders for { id, ... } data (used by
// compensations).
func (s *OrderModuleService) UpdateOrders(ctx context.Context, data []*dal.Record) error {
	if len(data) == 0 {
		return nil
	}
	_, err := s.Internal("Order").Update(ctx, toAny(data))
	return err
}
