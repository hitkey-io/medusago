package services

import (
	"context"
	"fmt"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	shippingmethod "github.com/hitkey-io/medusago/packages/core/utils/src/totals/shipping-method"
	"github.com/hitkey-io/medusago/packages/modules/order/src/utils"
)

func orNull(v any, ok bool) any {
	if !ok {
		return nil
	}
	return v
}

// CreateReturn mirrors createReturn (services/actions/create-return.ts):
// the return, its items and shipping method are created with a confirmed
// RETURN_REQUEST order change; the return is retrieved in the same entity
// manager (items, shipping methods with tax lines and adjustments).
func (s *OrderModuleService) CreateReturn(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	// One entity manager whose loaded collections are not refreshed: the
	// returned order graph is the one loaded before the change is applied
	// (upstream flushes the changes of confirmOrderChange at commit, after
	// the return is retrieved). order.summary is the exception: the JOINED
	// retrieve of the return hydrates it from its rows without a version
	// filter, so formatOrder reports the first summary row (version 1).
	ctx = dal.WithRefreshedRelations(dal.WithoutRelationRefresh(ctx), "order.summary")
	var out *dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		orderID := fmt.Sprint(get(data, "order_id"))
		f := dal.NewRecord()
		f.Set("id", orderID)
		orders, _, err := t.findOrders(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"items"}}, false)
		if err != nil {
			return err
		}
		if len(orders) == 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Order with id: %s was not found", orderID))
		}
		order := orders[0]

		returnID := common.GenerateEntityID("", "return")
		ret := dal.NewRecord()
		ret.Set("id", returnID)
		ret.Set("order_id", get(data, "order_id"))
		ret.Set("order_version", get(order, "version"))
		ret.Set("status", "requested")
		ret.Set("no_notification", get(data, "no_notification"))
		ret.Set("refund_amount", orNull(data.Get("refund_amount")))
		ret.Set("location_id", orNull(data.Get("location_id")))
		ret.Set("requested_at", orNull(data.Get("requested_at")))

		actions := []any{}
		var items []any
		for _, item := range records(get(data, "items")) {
			a := dal.NewRecord()
			a.Set("action", utils.ReturnItem)
			a.Set("return_id", returnID)
			setDefined(a, "internal_note", item, "internal_note")
			a.Set("reference", "return")
			a.Set("reference_id", returnID)
			d := dal.NewRecord()
			d.Set("reference_id", get(item, "id"))
			d.Set("quantity", get(item, "quantity"))
			setDefined(d, "metadata", item, "metadata")
			a.Set("details", d)
			actions = append(actions, a)

			ri := dal.NewRecord()
			ri.Set("reason_id", get(item, "reason_id"))
			ri.Set("return_id", returnID)
			ri.Set("item_id", get(item, "id"))
			ri.Set("quantity", get(item, "quantity"))
			ri.Set("note", get(item, "note"))
			ri.Set("metadata", get(item, "metadata"))
			items = append(items, ri)
		}
		if items == nil {
			items = []any{}
		}
		ret.Set("items", items)

		if sm, ok := data.Get("shipping_method"); ok && sm != nil {
			var methodID any
			if id, isString := sm.(string); isString {
				methodID = id
			} else {
				in := copyRecord(rec(sm))
				in.Set("order_id", get(data, "order_id"))
				in.Set("return_id", returnID)
				methods, err := t.CreateOrderShippingMethods(ctx, []*dal.Record{in})
				if err != nil {
					return err
				}
				methodID = get(methods[0], "id")
			}
			method, err := m.Internal("OrderShippingMethod").Retrieve(ctx, methodID, utilsmodulessdk.FindConfig{Relations: []string{"tax_lines", "adjustments"}})
			if err != nil {
				return err
			}
			total := shippingmethod.GetShippingMethodsTotals([]totals.Object{method}, false)[get(method, "id")].Get("total")
			a := dal.NewRecord()
			a.Set("action", utils.ShippingAdd)
			a.Set("reference", "order_shipping_method")
			a.Set("reference_id", methodID)
			a.Set("amount", total)
			d := dal.NewRecord()
			d.Set("order_id", get(ret, "order_id"))
			d.Set("return_id", returnID)
			a.Set("details", d)
			actions = append(actions, a)
		}

		c := dal.NewRecord()
		c.Set("order_id", get(data, "order_id"))
		c.Set("return_id", returnID)
		c.Set("change_type", "return_request")
		c.Set("reference", "return")
		c.Set("reference_id", returnID)
		for _, k := range []string{"description", "internal_note", "created_by", "metadata"} {
			c.Set(k, get(data, k))
		}
		c.Set("actions", actions)
		changes, err := t.createOrderChange(ctx, []*dal.Record{c})
		if err != nil {
			return err
		}
		if _, err := t.CreateReturns(ctx, []*dal.Record{ret}); err != nil {
			return err
		}
		if _, err := t.ConfirmOrderChange(ctx, []any{fmt.Sprint(get(changes[0], "id"))}); err != nil {
			return err
		}
		out, err = t.RetrieveRelated(ctx, "Return", returnID, utilsmodulessdk.FindConfig{
			Relations: []string{"items", "shipping_methods", "shipping_methods.tax_lines", "shipping_methods.adjustments"},
		})
		if err != nil {
			return err
		}
		if err := initializeOrderShippingTaxLines(ctx, m, out); err != nil {
			return err
		}
		return createdReturnShape(ctx, m, out)
	})
	return out, err
}

// createdReturnShape gives the retrieved return the shape of the returnRef
// upstream serializes (the entity em.create'd in the shared entity manager):
//   - order.transactions and order.credit_lines are initialized (loaded by
//     confirmOrderChange at the version before the change);
//   - the return shipping methods are the order_shipping rows created with
//     the shipping method (version = order_version), not the ones of the
//     confirmed version;
//   - the created return has its transactions collection and null
//     exchange_id / claim_id, and its created items no unit_price.
func createdReturnShape(ctx context.Context, m *utilsmodulessdk.MedusaService, ret *dal.Record) error {
	version := get(ret, "order_version")
	if order := rec(get(ret, "order")); order != nil {
		for _, rel := range []struct{ key, class string }{{"transactions", "OrderTransaction"}, {"credit_lines", "OrderCreditLine"}} {
			if order.Has(rel.key) {
				continue
			}
			f := dal.NewRecord()
			f.Set("order_id", get(order, "id"))
			f.Set("version", version)
			rows, err := m.Internal(rel.class).List(ctx, f, utilsmodulessdk.FindConfig{})
			if err != nil {
				return err
			}
			list := make([]any, 0, len(rows))
			for _, r := range rows {
				r.Delete("order")
				list = append(list, r)
			}
			order.Set(rel.key, list)
		}
	}
	for _, sm := range records(get(ret, "shipping_methods")) {
		detail := rec(get(sm, "detail"))
		if detail == nil {
			continue
		}
		f := dal.NewRecord()
		f.Set("return_id", get(ret, "id"))
		f.Set("shipping_method_id", get(sm, "id"))
		f.Set("version", version)
		rows, err := m.Internal("OrderShipping").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			for _, k := range []string{"id", "version", "created_at", "updated_at"} {
				if detail.Has(k) {
					v, _ := rows[0].Get(k)
					detail.Set(k, v)
				}
			}
		}
	}
	for _, item := range records(get(ret, "items")) {
		if v, ok := item.Get("unit_price"); ok && v == nil {
			item.Delete("unit_price")
		}
	}
	if !ret.Has("transactions") {
		keys := ret.Keys()
		values := make([]any, len(keys))
		for i, k := range keys {
			values[i], _ = ret.Get(k)
			ret.Delete(k)
		}
		for i, k := range keys {
			ret.Set(k, values[i])
			if k == "shipping_methods" {
				ret.Set("transactions", []any{})
				if !slicesContains(keys, "exchange_id") {
					ret.Set("exchange_id", nil)
				}
				if !slicesContains(keys, "claim_id") {
					ret.Set("claim_id", nil)
				}
			}
		}
	}
	return nil
}

func slicesContains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// initializeOrderShippingTaxLines mirrors the serialization of the
// initialized collections of the shared entity manager upstream:
// confirmOrderChange loaded the tax lines of the order shipping methods
// (the totals of the change), so return.order.shipping_methods[].
// shipping_method carries them (before its adjustments).
func initializeOrderShippingTaxLines(ctx context.Context, m *utilsmodulessdk.MedusaService, ret *dal.Record) error {
	order := rec(get(ret, "order"))
	var methods []*dal.Record
	var ids []any
	for _, os := range records(get(order, "shipping_methods")) {
		sm := rec(get(os, "shipping_method"))
		if sm == nil || sm.Has("tax_lines") {
			continue
		}
		methods = append(methods, sm)
		ids = append(ids, get(sm, "id"))
	}
	if len(methods) == 0 {
		return nil
	}
	f := dal.NewRecord()
	f.Set("shipping_method_id", ids)
	lines, err := m.Internal("OrderShippingMethodTaxLine").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	byMethod := map[string][]any{}
	for _, l := range lines {
		// the parent shipping method is an ancestor (circular reference)
		l.Delete("shipping_method")
		k := fmt.Sprint(get(l, "shipping_method_id"))
		byMethod[k] = append(byMethod[k], l)
	}
	for _, sm := range methods {
		taxLines := byMethod[fmt.Sprint(get(sm, "id"))]
		if taxLines == nil {
			taxLines = []any{}
		}
		keys := sm.Keys()
		values := make([]any, len(keys))
		for i, k := range keys {
			values[i], _ = sm.Get(k)
			sm.Delete(k)
		}
		inserted := false
		for i, k := range keys {
			if k == "adjustments" && !inserted {
				sm.Set("tax_lines", taxLines)
				inserted = true
			}
			sm.Set(k, values[i])
		}
		if !inserted {
			sm.Set("tax_lines", taxLines)
		}
	}
	return nil
}

// ReceiveReturn mirrors receiveReturn (services/actions/receive-return.ts)
// followed by the retrieval of the received return.
func (s *OrderModuleService) ReceiveReturn(ctx context.Context, data *dal.Record) (*dal.Record, error) {
	returnID := fmt.Sprint(get(data, "return_id"))
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		entry, err := t.RetrieveRelated(ctx, "Return", returnID, utilsmodulessdk.FindConfig{
			Select:    []string{"id", "order_id"},
			Relations: []string{"items", "items.item"},
		})
		if err != nil {
			return err
		}
		actions := []any{}
		var inputs []*dal.Record
		for _, item := range records(get(data, "items")) {
			a := dal.NewRecord()
			a.Set("action", utils.ReceiveReturnItem)
			a.Set("internal_note", get(item, "internal_note"))
			a.Set("reference", get(data, "reference"))
			a.Set("reference_id", get(data, "reference_id"))
			d := dal.NewRecord()
			d.Set("reference_id", get(item, "id"))
			d.Set("quantity", get(item, "quantity"))
			a.Set("details", d)
			actions = append(actions, a)
			inputs = append(inputs, a)
		}
		c := dal.NewRecord()
		c.Set("order_id", get(entry, "order_id"))
		c.Set("return_id", get(entry, "id"))
		c.Set("reference", "return")
		c.Set("reference_id", get(entry, "id"))
		c.Set("change_type", "return_receive")
		for _, k := range []string{"description", "internal_note", "created_by", "metadata"} {
			c.Set(k, get(data, k))
		}
		c.Set("actions", actions)
		changes, err := t.createOrderChange(ctx, []*dal.Record{c})
		if err != nil {
			return err
		}
		if _, err := t.ConfirmOrderChange(ctx, []any{fmt.Sprint(get(changes[0], "id"))}); err != nil {
			return err
		}
		var updates []any
		allReceived := true
		for _, item := range records(get(entry, "items")) {
			for _, a := range inputs {
				d := rec(get(a, "details"))
				if get(d, "reference_id") != get(item, "item_id") {
					continue
				}
				received := totals.Add(totalsOrZero(get(item, "received_quantity")), get(d, "quantity"))
				item.Set("received_quantity", received)
				u := dal.NewRecord()
				u.Set("id", get(item, "id"))
				u.Set("received_quantity", received)
				updates = append(updates, u)
				break
			}
			if !totals.Eq(get(item, "received_quantity"), get(item, "quantity")) {
				allReceived = false
			}
		}
		rd := dal.NewRecord()
		if allReceived {
			rd.Set("status", "received")
			rd.Set("received_at", time.Now())
		} else {
			rd.Set("status", "partially_received")
		}
		sel := dal.NewRecord()
		sel.Set("id", get(entry, "id"))
		upd := dal.NewRecord()
		upd.Set("selector", sel)
		upd.Set("data", rd)
		if _, err := m.Internal("Return").Update(ctx, []any{upd}); err != nil {
			return err
		}
		if len(updates) > 0 {
			if _, err := m.Internal("ReturnItem").Update(ctx, updates); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.RetrieveRelated(ctx, "Return", returnID, utilsmodulessdk.FindConfig{
		Relations: []string{"items", "items.item", "shipping_methods", "shipping_methods.tax_lines", "shipping_methods.adjustments"},
	})
}

func totalsOrZero(v any) any {
	if v == nil {
		return 0
	}
	return v
}

func rec(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}
