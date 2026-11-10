package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	carttotals "github.com/hitkey-io/medusago/packages/core/utils/src/totals/cart"
	"github.com/hitkey-io/medusago/packages/modules/order/src/utils"
)

func invalidData(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, msg)
}

func (s *OrderModuleService) tx(m *utilsmodulessdk.MedusaService) *OrderModuleService {
	return &OrderModuleService{MedusaService: m}
}

func asRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

// createOrderChange mirrors createOrderChange_.
func (s *OrderModuleService) createOrderChange(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var orderIDs []any
	for _, d := range data {
		orderIDs = append(orderIDs, get(d, "order_id"))
	}
	f := dal.NewRecord()
	f.Set("order_id", orderIDs)
	f.Set("status", []any{"pending", "requested"})
	existing, err := s.Internal("OrderChange").List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	existingByOrder := map[string]bool{}
	for _, c := range existing {
		existingByOrder[fmt.Sprint(get(c, "order_id"))] = true
	}
	dataMap := map[string]*dal.Record{}
	for _, d := range data {
		dataMap[fmt.Sprint(get(d, "order_id"))] = d
	}
	of := dal.NewRecord()
	of.Set("id", orderIDs)
	orders, err := s.Internal("Order").List(ctx, of, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
	if err != nil {
		return nil, err
	}
	if len(orders) != len(orderIDs) {
		found := map[string]bool{}
		for _, o := range orders {
			found[fmt.Sprint(get(o, "id"))] = true
		}
		var missing []string
		for _, id := range orderIDs {
			if !found[fmt.Sprint(id)] {
				missing = append(missing, fmt.Sprint(id))
			}
		}
		return nil, invalidData("Order could not be found: " + strings.Join(missing, ", "))
	}
	var input []*dal.Record
	for _, o := range orders {
		id := fmt.Sprint(get(o, "id"))
		if existingByOrder[id] {
			return nil, invalidData(fmt.Sprintf("Order (%s) already has an existing active order change", id))
		}
		c := copyRecord(dataMap[id])
		version := versionOf(get(o, "version")) + 1
		c.Set("version", version)
		if acts := records(get(c, "actions")); acts != nil {
			var out []any
			for _, a := range acts {
				out = append(out, initChangeAction(copyRecord(a), c))
			}
			c.Set("actions", out)
		}
		input = append(input, c)
	}
	return s.Internal("OrderChange").Create(ctx, input)
}

func versionOf(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	}
	return 0
}

// initChangeAction mirrors the OrderChangeAction onInit hook for actions
// created with their order change.
func initChangeAction(a, change *dal.Record) *dal.Record {
	setIfNil := func(k string, v any) {
		if get(a, k) == nil {
			a.Set(k, v)
		}
	}
	setIfNil("version", get(change, "version"))
	setIfNil("order_id", get(change, "order_id"))
	setIfNil("claim_id", get(change, "claim_id"))
	setIfNil("exchange_id", get(change, "exchange_id"))
	if get(a, "claim_id") == nil && get(a, "exchange_id") == nil {
		a.Set("return_id", coalesce(get(a, "return_id"), get(change, "return_id")))
	}
	return a
}

func coalesce(vs ...any) any {
	for _, v := range vs {
		if v != nil {
			return v
		}
	}
	return nil
}

// CreateOrderChange mirrors createOrderChange (array input).
func (s *OrderModuleService) CreateOrderChange(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		var err error
		out, err = s.tx(m).createOrderChange(ctx, data)
		return err
	})
	return out, err
}

// getAndValidateOrderChange mirrors getAndValidateOrderChange_.
func (s *OrderModuleService) getAndValidateOrderChange(ctx context.Context, ids []any, includeActions bool) ([]*dal.Record, error) {
	ids = dedupeAny(ids)
	sel := []string{"id", "order_id", "return_id", "claim_id", "exchange_id", "version", "status"}
	var rels []string
	var order []dal.OrderBy
	if includeActions {
		sel = append(sel, "actions")
		rels = append(rels, "actions")
		order = append(order, dal.OrderBy{Field: "actions.ordering", Direction: "ASC"})
	}
	f := dal.NewRecord()
	f.Set("id", ids)
	changes, err := s.MedusaService.ListBySuffix(ctx, "OrderChanges", f, utilsmodulessdk.FindConfig{Select: sel, Relations: rels, Order: order})
	if err != nil {
		return nil, err
	}
	if len(changes) != len(ids) {
		found := map[string]bool{}
		for _, c := range changes {
			found[fmt.Sprint(get(c, "id"))] = true
		}
		var missing []string
		for _, id := range ids {
			if !found[fmt.Sprint(id)] {
				missing = append(missing, fmt.Sprint(id))
			}
		}
		return nil, invalidData("Order Change could not be found: " + strings.Join(missing, ", "))
	}
	for _, c := range changes {
		if st := get(c, "status"); st != "pending" && st != "requested" {
			return nil, invalidData(fmt.Sprintf("Order Change cannot be modified: %v.", get(c, "id")))
		}
	}
	return changes, nil
}

func dedupeAny(list []any) []any {
	seen := map[string]bool{}
	var out []any
	for _, x := range list {
		k := fmt.Sprint(x)
		if !seen[k] {
			seen[k] = true
			out = append(out, x)
		}
	}
	return out
}

func changeInputs(data []any) ([]any, []*dal.Record) {
	var ids []any
	var updates []*dal.Record
	for _, d := range data {
		if id, ok := d.(string); ok {
			ids = append(ids, id)
			u := dal.NewRecord()
			u.Set("id", id)
			updates = append(updates, u)
			continue
		}
		r := copyRecord(asRecord(d))
		ids = append(ids, get(r, "id"))
		updates = append(updates, r)
	}
	return ids, updates
}

func (s *OrderModuleService) setChangeStatus(ctx context.Context, data []any, includeActions bool, dateField, status string) ([]*dal.Record, error) {
	ids, updates := changeInputs(data)
	changes, err := s.getAndValidateOrderChange(ctx, ids, includeActions)
	if err != nil {
		return nil, err
	}
	var u []any
	for _, up := range updates {
		up.Set(dateField, dal.Now())
		up.Set("status", status)
		u = append(u, up)
	}
	if _, err := s.Internal("OrderChange").Update(ctx, u); err != nil {
		return nil, err
	}
	return changes, nil
}

// CancelOrderChange mirrors cancelOrderChange (ids or { id, ... } data).
func (s *OrderModuleService) CancelOrderChange(ctx context.Context, data []any) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		_, err := s.tx(m).setChangeStatus(ctx, data, false, "canceled_at", "canceled")
		return err
	})
}

// DeclineOrderChange mirrors declineOrderChange.
func (s *OrderModuleService) DeclineOrderChange(ctx context.Context, data []any) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		_, err := s.tx(m).setChangeStatus(ctx, data, false, "declined_at", "declined")
		return err
	})
}

// OrderChangeReturn mirrors OrderChangeReturn.
type OrderChangeReturn struct {
	Items           []*dal.Record
	ShippingMethods []*dal.Record
	CreditLines     []*dal.Record
}

// ConfirmOrderChange mirrors confirmOrderChange.
func (s *OrderModuleService) ConfirmOrderChange(ctx context.Context, data []any) (*OrderChangeReturn, error) {
	var out *OrderChangeReturn
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		changes, err := t.setChangeStatus(ctx, data, true, "confirmed_at", "confirmed")
		if err != nil {
			return err
		}
		var actions []*dal.Record
		for _, c := range changes {
			actions = append(actions, records(get(c, "actions"))...)
		}
		out, err = t.applyOrderChanges(ctx, actions)
		return err
	})
	return out, err
}

// applyOrderChanges mirrors applyOrderChanges_.
func (s *OrderModuleService) applyOrderChanges(ctx context.Context, changeActions []*dal.Record) (*OrderChangeReturn, error) {
	actionsMap := map[string][]*dal.Record{}
	var orderIDs []any
	var usedActions []any
	for _, a := range changeActions {
		if b, _ := get(a, "applied").(bool); b {
			continue
		}
		id := fmt.Sprint(get(a, "order_id"))
		orderIDs = append(orderIDs, get(a, "order_id"))
		actionsMap[id] = append(actionsMap[id], a)
		sel := dal.NewRecord()
		sel.Set("id", get(a, "id"))
		d := dal.NewRecord()
		d.Set("applied", true)
		u := dal.NewRecord()
		u.Set("selector", sel)
		u.Set("data", d)
		usedActions = append(usedActions, u)
	}
	if len(orderIDs) == 0 {
		return &OrderChangeReturn{Items: []*dal.Record{}, ShippingMethods: []*dal.Record{}, CreditLines: []*dal.Record{}}, nil
	}
	f := dal.NewRecord()
	f.Set("id", dedupeAny(orderIDs))
	orders, _, err := s.listOrders(dal.WithSharedIdentityMap(ctx), f, utilsmodulessdk.FindConfig{
		Select:    []string{"id", "version", "items.detail", "summary", "total", "email"},
		Relations: []string{"transactions", "credit_lines"},
	}, false)
	if err != nil {
		return nil, err
	}
	res, err := utils.ApplyChangesToOrder(orders, actionsMap, utils.ProcessOptions{AddActionReferenceToObject: true},
		func(order *dal.Record, items, sms, ladj, sadj []*dal.Record) error {
			return s.includeTaxLinesAndAdjustmentsToPreview(ctx, order, items, sms, ladj, sadj)
		})
	if err != nil {
		return nil, err
	}
	out := &OrderChangeReturn{}
	if len(res.OrderToUpdate) > 0 {
		if _, err := s.Internal("Order").Update(ctx, toAny(res.OrderToUpdate)); err != nil {
			return nil, err
		}
	}
	if len(usedActions) > 0 {
		if _, err := s.Internal("OrderChangeAction").Update(ctx, usedActions); err != nil {
			return nil, err
		}
	}
	if len(res.ItemsToUpsert) > 0 {
		if out.Items, err = s.Internal("OrderItem").Upsert(ctx, res.ItemsToUpsert); err != nil {
			return nil, err
		}
	}
	if len(res.SummariesToUpsert) > 0 {
		if _, err := s.Internal("OrderSummary").Upsert(ctx, res.SummariesToUpsert); err != nil {
			return nil, err
		}
	}
	if len(res.ShippingMethodsToUpsert) > 0 {
		if out.ShippingMethods, err = s.Internal("OrderShipping").Upsert(ctx, res.ShippingMethodsToUpsert); err != nil {
			return nil, err
		}
	}
	if len(res.CreditLinesToUpsert) > 0 {
		if out.CreditLines, err = s.Internal("OrderCreditLine").Upsert(ctx, res.CreditLinesToUpsert); err != nil {
			return nil, err
		}
	}
	if len(res.LineItemAdjustmentsToCreate) > 0 {
		if _, err := s.Internal("OrderLineItemAdjustment").Create(ctx, res.LineItemAdjustmentsToCreate); err != nil {
			return nil, err
		}
	}
	if len(res.ShippingMethodAdjustmentsToCreate) > 0 {
		if _, err := s.Internal("OrderShippingMethodAdjustment").Create(ctx, res.ShippingMethodAdjustmentsToCreate); err != nil {
			return nil, err
		}
	}
	if out.Items == nil {
		out.Items = []*dal.Record{}
	}
	if out.ShippingMethods == nil {
		out.ShippingMethods = []*dal.Record{}
	}
	if out.CreditLines == nil {
		out.CreditLines = []*dal.Record{}
	}
	return out, nil
}

// includeTaxLinesAndAdjustmentsToPreview mirrors
// includeTaxLinesAndAdjustmentsToPreview.
func (s *OrderModuleService) includeTaxLinesAndAdjustmentsToPreview(ctx context.Context, order *dal.Record, itemsToUpsert, shippingMethodsToUpsert, lineItemAdjustmentsToCreate, shippingMethodAdjustmentsToCreate []*dal.Record) error {
	added := map[string]bool{}
	var addedIDs []any
	items := records(get(order, "items"))
	for _, item := range items {
		detail := asRecord(get(item, "detail"))
		if detail == nil || get(item, "id") != get(detail, "item_id") {
			added[fmt.Sprint(get(item, "id"))] = true
			addedIDs = append(addedIDs, get(item, "id"))
		}
	}
	addedSM := map[string]bool{}
	var addedSMIDs []any
	sms := records(get(order, "shipping_methods"))
	for _, sm := range sms {
		if get(sm, "shipping_option_id") == nil {
			addedSM[fmt.Sprint(get(sm, "id"))] = true
			addedSMIDs = append(addedSMIDs, get(sm, "id"))
		}
	}
	if len(addedIDs) > 0 {
		f := dal.NewRecord()
		f.Set("id", addedIDs)
		details, err := s.MedusaService.ListBySuffix(ctx, "OrderLineItems", f, utilsmodulessdk.FindConfig{Relations: []string{"adjustments", "tax_lines"}})
		if err != nil {
			return err
		}
		out := make([]any, len(items))
		for idx, item := range items {
			out[idx] = item
			if !added[fmt.Sprint(get(item, "id"))] {
				continue
			}
			var lineItem *dal.Record
			for _, d := range details {
				if get(d, "id") == get(item, "id") {
					lineItem = copyRecord(d)
				}
			}
			actions := get(item, "actions")
			item.Delete("actions")
			var newItem *dal.Record
			for _, u := range itemsToUpsert {
				if get(u, "item_id") == get(item, "id") {
					newItem = u
					break
				}
			}
			var adjustments []any
			for _, a := range lineItemAdjustmentsToCreate {
				if get(a, "item_id") == get(newItem, "item_id") {
					adjustments = append(adjustments, a)
				}
			}
			unitPrice := coalesce(get(newItem, "unit_price"), get(item, "unit_price"))
			compareAt := coalesce(get(newItem, "compare_at_unit_price"), get(item, "compare_at_unit_price"))
			lineItem.Delete("raw_unit_price")
			lineItem.Delete("raw_compare_at_unit_price")
			lineItem.Set("actions", actions)
			lineItem.Set("quantity", get(newItem, "quantity"))
			lineItem.Set("unit_price", unitPrice)
			if !truthyValue(compareAt) {
				compareAt = nil
			}
			lineItem.Set("compare_at_unit_price", compareAt)
			if adjustments == nil {
				adjustments = []any{}
			}
			lineItem.Set("adjustments", adjustments)
			detail := copyRecord(newItem)
			for _, k := range item.Keys() {
				v, _ := item.Get(k)
				detail.Set(k, v)
			}
			lineItem.Set("detail", detail)
			out[idx] = lineItem
		}
		order.Set("items", out)
	}
	if len(addedSMIDs) > 0 {
		f := dal.NewRecord()
		f.Set("id", addedSMIDs)
		details, err := s.MedusaService.ListBySuffix(ctx, "OrderShippingMethods", f, utilsmodulessdk.FindConfig{Relations: []string{"adjustments", "tax_lines"}})
		if err != nil {
			return err
		}
		out := make([]any, len(sms))
		for idx, sm := range sms {
			out[idx] = sm
			if !addedSM[fmt.Sprint(get(sm, "id"))] {
				continue
			}
			var shippingMethod *dal.Record
			for _, d := range details {
				if get(d, "id") == get(sm, "id") {
					shippingMethod = copyRecord(d)
				}
			}
			actions := get(sm, "actions")
			sm.Delete("actions")
			var newItem *dal.Record
			for _, u := range shippingMethodsToUpsert {
				if get(u, "id") == get(sm, "id") {
					newItem = u
					break
				}
			}
			var adjustments []any
			for _, a := range shippingMethodAdjustmentsToCreate {
				if get(a, "shipping_method_id") == get(sm, "id") {
					adjustments = append(adjustments, a)
				}
			}
			sm.Set("shipping_method_id", get(sm, "id"))
			sm.Delete("id")
			if shippingMethod == nil {
				shippingMethod = dal.NewRecord()
			}
			shippingMethod.Set("actions", actions)
			if adjustments == nil {
				adjustments = []any{}
			}
			shippingMethod.Set("adjustments", adjustments)
			detail := copyRecord(sm)
			if newItem != nil {
				for _, k := range newItem.Keys() {
					v, _ := newItem.Get(k)
					detail.Set(k, v)
				}
			}
			shippingMethod.Set("detail", detail)
			out[idx] = shippingMethod
		}
		order.Set("shipping_methods", out)
	}
	return nil
}

func truthyValue(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case float64:
		return t != 0
	case int:
		return t != 0
	case string:
		return t != ""
	case bool:
		return t
	case *totals.BN:
		return !t.IsZero()
	case *totals.BigNumber:
		return t.Numeric() != 0
	}
	return true
}

// PreviewOrderChange mirrors previewOrderChange.
func (s *OrderModuleService) PreviewOrderChange(ctx context.Context, orderID string) (*dal.Record, error) {
	ctx = dal.WithSharedIdentityMap(ctx)
	order, err := s.RetrieveOrder(ctx, orderID, utilsmodulessdk.FindConfig{
		Select:    []string{"id", "version", "items.detail", "summary", "total", "email"},
		Relations: []string{"transactions", "credit_lines"},
	})
	if err != nil {
		return nil, err
	}
	change := asRecord(get(order, "order_change"))
	if change == nil {
		return order, nil
	}
	full, err := s.Internal("OrderChange").Retrieve(ctx, get(change, "id"), utilsmodulessdk.FindConfig{Relations: []string{"actions"}})
	if err != nil {
		return nil, err
	}
	actions := records(get(full, "actions"))
	sortByOrdering(actions)
	res, err := utils.ApplyChangesToOrder([]*dal.Record{order}, map[string][]*dal.Record{orderID: actions}, utils.ProcessOptions{AddActionReferenceToObject: true}, nil)
	if err != nil {
		return nil, err
	}
	calculated := res.CalculatedOrders[orderID]
	if err := s.includeTaxLinesAndAdjustmentsToPreview(ctx, calculated.Order, res.ItemsToUpsert, res.ShippingMethodsToUpsert, res.LineItemAdjustmentsToCreate, res.ShippingMethodAdjustmentsToCreate); err != nil {
		return nil, err
	}
	calcOrder := calculated.Order
	carttotals.DecorateCartTotals(calcOrder, false)
	calcOrder.Set("summary", calculated.Instance.GetSummaryFromOrder(calcOrder))
	totals.CreateRawPropertiesFromBigNumber(calcOrder, func() totals.Object { return dal.NewRecord() })
	return calcOrder, nil
}

func sortByOrdering(list []*dal.Record) {
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && versionOf(get(list[j], "ordering")) < versionOf(get(list[j-1], "ordering")); j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

// registerChange creates an order change with actions and confirms it
// (services/actions/register-*.ts).
func (s *OrderModuleService) registerChange(ctx context.Context, data *dal.Record, actions []any) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		c := dal.NewRecord()
		for _, k := range []string{"order_id", "description", "internal_note", "created_by", "metadata"} {
			c.Set(k, get(data, k))
		}
		c.Set("actions", actions)
		changes, err := t.createOrderChange(ctx, []*dal.Record{c})
		if err != nil {
			return err
		}
		_, err = t.ConfirmOrderChange(ctx, []any{fmt.Sprint(get(changes[0], "id"))})
		return err
	})
}

func itemActions(data *dal.Record, action string, extra func(a *dal.Record)) []any {
	var out []any
	for _, item := range records(get(data, "items")) {
		a := dal.NewRecord()
		a.Set("action", action)
		setDefined(a, "internal_note", item, "internal_note")
		setDefined(a, "reference", data, "reference")
		setDefined(a, "reference_id", data, "reference_id")
		if extra != nil {
			extra(a)
		}
		d := dal.NewRecord()
		setDefined(d, "reference_id", item, "id")
		setDefined(d, "quantity", item, "quantity")
		setDefined(d, "metadata", item, "metadata")
		a.Set("details", d)
		out = append(out, a)
	}
	return out
}

// RegisterFulfillment mirrors registerFulfillment.
func (s *OrderModuleService) RegisterFulfillment(ctx context.Context, data *dal.Record) error {
	return s.registerChange(ctx, data, itemActions(data, utils.FulfillItem, nil))
}

// RegisterShipment mirrors registerShipment.
func (s *OrderModuleService) RegisterShipment(ctx context.Context, data *dal.Record) error {
	return s.registerChange(ctx, data, itemActions(data, utils.ShipItem, nil))
}

// RegisterDelivery mirrors registerDelivery.
func (s *OrderModuleService) RegisterDelivery(ctx context.Context, data *dal.Record) error {
	return s.registerChange(ctx, data, itemActions(data, utils.DeliverItem, nil))
}

// CancelFulfillment mirrors cancelFulfillment.
func (s *OrderModuleService) CancelFulfillment(ctx context.Context, data *dal.Record) error {
	return s.registerChange(ctx, data, itemActions(data, utils.CancelItemFulfillment, func(a *dal.Record) {
		setDefined(a, "claim_id", data, "claim_id")
		setDefined(a, "exchange_id", data, "exchange_id")
	}))
}

// RegisterOrderChange mirrors registerOrderChange.
func (s *OrderModuleService) RegisterOrderChange(ctx context.Context, data []*dal.Record) ([]*dal.Record, error) {
	var out []*dal.Record
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		var ids []any
		for _, d := range data {
			ids = append(ids, get(d, "order_id"))
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		orders, err := m.Internal("Order").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
		if err != nil {
			return err
		}
		versions := map[string]any{}
		for _, o := range orders {
			versions[fmt.Sprint(get(o, "id"))] = get(o, "version")
		}
		var input []*dal.Record
		for _, d := range data {
			v := versions[fmt.Sprint(get(d, "order_id"))]
			c := dal.NewRecord()
			for _, k := range []string{"order_id", "change_type", "internal_note", "description", "metadata"} {
				c.Set(k, get(d, k))
			}
			c.Set("confirmed_at", dal.Now())
			c.Set("created_by", get(d, "created_by"))
			c.Set("confirmed_by", get(d, "confirmed_by"))
			c.Set("status", "confirmed")
			c.Set("version", v)
			a := dal.NewRecord()
			a.Set("action", utils.UpdateOrderProperties)
			a.Set("details", get(d, "details"))
			a.Set("version", v)
			a.Set("applied", true)
			c.Set("actions", []any{initChangeAction(a, c)})
			input = append(input, c)
		}
		out, err = m.Internal("OrderChange").Create(ctx, input)
		return err
	})
	return out, err
}

// ApplyPendingOrderActions mirrors applyPendingOrderActions.
func (s *OrderModuleService) ApplyPendingOrderActions(ctx context.Context, orderIDs []any) (*OrderChangeReturn, error) {
	var out *OrderChangeReturn
	err := s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		t := s.tx(m)
		f := dal.NewRecord()
		f.Set("id", orderIDs)
		orders, err := m.Internal("Order").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "version"}})
		if err != nil {
			return err
		}
		var ids []any
		for _, o := range orders {
			ids = append(ids, get(o, "id"))
		}
		af := dal.NewRecord()
		af.Set("order_id", ids)
		if len(orders) > 0 {
			af.Set("version", get(orders[0], "version"))
		}
		af.Set("applied", false)
		changes, err := m.Internal("OrderChangeAction").List(ctx, af, utilsmodulessdk.FindConfig{
			Select: []string{"id", "order_id", "return_id", "exchange_id", "claim_id", "ordering", "version", "applied", "reference", "reference_id", "action", "details", "amount", "raw_amount", "internal_note"},
			Order:  []dal.OrderBy{{Field: "ordering", Direction: "ASC"}},
		})
		if err != nil {
			return err
		}
		out, err = t.applyOrderChanges(ctx, changes)
		return err
	})
	return out, err
}

// setDefined copies a key only when it is present (JS undefined keys are
// dropped by JSON).
func setDefined(dst *dal.Record, key string, src *dal.Record, srcKey string) {
	if src == nil {
		return
	}
	if v, ok := src.Get(srcKey); ok {
		dst.Set(key, v)
	}
}

// ConfirmOrderChangeRecords confirms changes given as ids or { id, ... }.
func (s *OrderModuleService) ConfirmOrderChangeRecords(ctx context.Context, data []any) error {
	_, err := s.ConfirmOrderChange(ctx, data)
	return err
}
