package utils

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	carttotals "github.com/hitkey-io/medusago/packages/core/utils/src/totals/cart"
)

// ApplyChangesResult mirrors the result of applyChangesToOrder.
type ApplyChangesResult struct {
	LineItemAdjustmentsToCreate       []*dal.Record
	ShippingMethodAdjustmentsToCreate []*dal.Record
	ItemsToUpsert                     []*dal.Record
	CreditLinesToUpsert               []*dal.Record
	ShippingMethodsToUpsert           []*dal.Record
	SummariesToUpsert                 []*dal.Record
	OrderToUpdate                     []*dal.Record
	CalculatedOrders                  map[string]*CalculatedOrderChange
}

// IncludeTaxLinesFunc mirrors includeTaxLinesAndAdjustmentsToPreview.
type IncludeTaxLinesFunc func(order *dal.Record, itemsToUpsert, shippingMethodsToUpsert, lineItemAdjustmentsToCreate, shippingMethodAdjustmentsToCreate []*dal.Record) error

func newObject() totals.Object { return dal.NewRecord() }

// versionNumber returns a JSON number version as an int.
func versionNumber(v any) int {
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

// ApplyChangesToOrder mirrors applyChangesToOrder.
func ApplyChangesToOrder(orders []*dal.Record, actionsMap map[string][]*dal.Record, options ProcessOptions, includeTaxLines IncludeTaxLinesFunc) (*ApplyChangesResult, error) {
	res := &ApplyChangesResult{CalculatedOrders: map[string]*CalculatedOrderChange{}}
	editable := []string{"customer_id", "sales_channel_id", "email", "no_notification"}
	for _, order := range orders {
		orderID := fmt.Sprint(get(order, "id"))
		actions := actionsMap[orderID]
		transactions := records(get(order, "transactions"))
		if transactions == nil {
			transactions = []*dal.Record{}
		}
		calculated, err := CalculateOrderChange(order, transactions, actions, options)
		if err != nil {
			return nil, err
		}
		totals.CreateRawPropertiesFromBigNumber(calculated.Order, newObject)
		totals.CreateRawPropertiesFromBigNumber(calculated.Summary, newObject)

		orderVersion := versionNumber(get(order, "version"))
		version := orderVersion
		if len(actions) > 0 && get(actions[0], "version") != nil {
			version = versionNumber(get(actions[0], "version"))
		}
		attrs := dal.NewRecord()
		for _, attr := range editable {
			if !sameJSON(get(order, attr), get(calculated.Order, attr)) {
				attrs.Set(attr, get(calculated.Order, attr))
			}
		}

		for _, item := range records(get(calculated.Order, "items")) {
			if totals.Lte(get(item, "quantity"), 0) {
				continue
			}
			detail := asRecord(get(item, "detail"))
			isExisting := detail != nil && get(item, "id") == get(detail, "item_id")
			orderItem := item
			itemID := get(item, "id")
			if isExisting {
				orderItem = detail
				itemID = get(orderItem, "item_id")
			}
			u := dal.NewRecord()
			if versionNumber(get(orderItem, "version")) == version && get(orderItem, "version") != nil {
				u.Set("id", get(orderItem, "id"))
			}
			u.Set("item_id", itemID)
			u.Set("order_id", get(order, "id"))
			u.Set("version", version)
			u.Set("quantity", get(orderItem, "quantity"))
			u.Set("unit_price", coalesceAny(get(item, "unit_price"), get(orderItem, "unit_price")))
			u.Set("compare_at_unit_price", coalesceAny(get(item, "compare_at_unit_price"), get(orderItem, "compare_at_unit_price")))
			for _, k := range []string{"fulfilled_quantity", "delivered_quantity", "shipped_quantity", "return_requested_quantity", "return_received_quantity", "return_dismissed_quantity", "written_off_quantity"} {
				u.Set(k, orZero(get(orderItem, k)))
			}
			u.Set("metadata", get(orderItem, "metadata"))
			if version > orderVersion {
				for _, adj := range records(get(item, "adjustments")) {
					a := dal.NewRecord()
					a.Set("item_id", itemID)
					a.Set("version", version)
					a.Set("amount", get(adj, "amount"))
					a.Set("description", get(adj, "description"))
					a.Set("promotion_id", get(adj, "promotion_id"))
					a.Set("code", get(adj, "code"))
					a.Set("is_tax_inclusive", get(adj, "is_tax_inclusive"))
					res.LineItemAdjustmentsToCreate = append(res.LineItemAdjustmentsToCreate, a)
				}
			}
			res.ItemsToUpsert = append(res.ItemsToUpsert, u)
		}

		if version > orderVersion {
			for _, cl := range records(get(calculated.Order, "credit_lines")) {
				isExisting := get(cl, "id") != nil
				u := dal.NewRecord()
				if versionNumber(get(cl, "version")) == version && get(cl, "version") != nil {
					u.Set("id", get(cl, "id"))
				}
				u.Set("order_id", get(order, "id"))
				u.Set("version", version)
				u.Set("reference", get(cl, "reference"))
				u.Set("reference_id", get(cl, "reference_id"))
				u.Set("amount", get(cl, "amount"))
				u.Set("raw_amount", get(cl, "raw_amount"))
				if isExisting {
					u.Set("metadata", get(cl, "metadata"))
				} else {
					m := copyRecord(asRecord(get(cl, "metadata")))
					m.Set("created_in_version", version)
					u.Set("metadata", m)
				}
				res.CreditLinesToUpsert = append(res.CreditLinesToUpsert, u)
			}

			for _, sm := range records(get(calculated.Order, "shipping_methods")) {
				detail := asRecord(get(sm, "detail"))
				isNew := detail == nil
				var associated any
				hasShippingMethod := false
				if isNew {
					for _, a := range records(get(sm, "actions")) {
						if get(a, "action") == ShippingAdd && truthy(get(a, "reference_id")) {
							associated = a
							break
						}
					}
					hasShippingMethod = associated != nil
				} else {
					associated = get(detail, "shipping_method_id")
				}
				var u *dal.Record
				if isNew {
					u = copyRecord(sm)
				} else {
					u = copyRecord(detail)
				}
				u.Set("version", version)
				u.Set("shipping_method_id", associated)
				u.Delete("id")
				if !hasShippingMethod {
					res.ShippingMethodsToUpsert = append(res.ShippingMethodsToUpsert, u)
				}
				for _, adj := range records(get(sm, "adjustments")) {
					a := dal.NewRecord()
					a.Set("shipping_method_id", associated)
					a.Set("version", version)
					a.Set("amount", get(adj, "amount"))
					a.Set("description", get(adj, "description"))
					a.Set("promotion_id", get(adj, "promotion_id"))
					a.Set("code", get(adj, "code"))
					res.ShippingMethodAdjustmentsToCreate = append(res.ShippingMethodAdjustmentsToCreate, a)
				}
			}
			attrs.Set("version", version)
		}

		if includeTaxLines != nil {
			if err := includeTaxLines(calculated.Order, res.ItemsToUpsert, res.ShippingMethodsToUpsert, res.LineItemAdjustmentsToCreate, res.ShippingMethodAdjustmentsToCreate); err != nil {
				return nil, err
			}
			carttotals.DecorateCartTotals(calculated.Order, false)
		}

		summary := asRecord(get(order, "summary"))
		upsert := dal.NewRecord()
		if summary != nil && versionNumber(get(summary, "version")) == version && get(summary, "version") != nil {
			upsert.Set("id", get(summary, "id"))
		}
		upsert.Set("order_id", get(order, "id"))
		upsert.Set("version", version)
		upsert.Set("totals", calculated.Instance.GetSummaryFromOrder(calculated.Order))
		totals.CreateRawPropertiesFromBigNumber(upsert, newObject)
		res.SummariesToUpsert = append(res.SummariesToUpsert, upsert)

		if attrs.Len() > 0 {
			sel := dal.NewRecord()
			sel.Set("id", get(order, "id"))
			u := dal.NewRecord()
			u.Set("selector", sel)
			u.Set("data", attrs)
			res.OrderToUpdate = append(res.OrderToUpdate, u)
		}
		res.CalculatedOrders[orderID] = calculated
	}
	return res, nil
}

// sameJSON mirrors `a !== b` for JSON scalars (strings, booleans, null).
func sameJSON(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}
