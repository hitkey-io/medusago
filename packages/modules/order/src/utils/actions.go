package utils

import (
	"fmt"
	"strconv"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// ChangeActionType values mirror ChangeActionType.
const (
	CancelItemFulfillment      = "CANCEL_ITEM_FULFILLMENT"
	CancelReturnItem           = "CANCEL_RETURN_ITEM"
	UpdateOrderProperties      = "UPDATE_ORDER_PROPERTIES"
	CreditLineAdd              = "CREDIT_LINE_ADD"
	DeliverItem                = "DELIVER_ITEM"
	FulfillItem                = "FULFILL_ITEM"
	ItemAdd                    = "ITEM_ADD"
	ItemAdjustmentsReplace     = "ITEM_ADJUSTMENTS_REPLACE"
	ItemRemove                 = "ITEM_REMOVE"
	ItemUpdate                 = "ITEM_UPDATE"
	PromotionAdd               = "PROMOTION_ADD"
	PromotionRemove            = "PROMOTION_REMOVE"
	ReceiveDamagedReturnItem   = "RECEIVE_DAMAGED_RETURN_ITEM"
	ReceiveReturnItem          = "RECEIVE_RETURN_ITEM"
	ReinstateItem              = "REINSTATE_ITEM"
	ReturnItem                 = "RETURN_ITEM"
	ShipItem                   = "SHIP_ITEM"
	ShippingAdd                = "SHIPPING_ADD"
	ShippingAdjustmentsReplace = "SHIPPING_ADJUSTMENTS_REPLACE"
	ShippingRemove             = "SHIPPING_REMOVE"
	ShippingUpdate             = "SHIPPING_UPDATE"
	TransferCustomer           = "TRANSFER_CUSTOMER"
	WriteOffItem               = "WRITE_OFF_ITEM"
)

func setActionReference(existing, action *dal.Record, options ProcessOptions) {
	if options.AddActionReferenceToObject {
		list, _ := get(existing, "actions").([]any)
		existing.Set("actions", append(list, action))
	}
}

func details(action *dal.Record) *dal.Record { return asRecord(get(action, "details")) }

func detailOf(item *dal.Record) *dal.Record {
	d := asRecord(get(item, "detail"))
	if d == nil {
		d = dal.NewRecord()
		item.Set("detail", d)
	}
	return d
}

func findItem2(order *dal.Record, id any) (*dal.Record, int) {
	for i, item := range records(get(order, "items")) {
		if get(item, "id") == id {
			return item, i
		}
	}
	return nil, -1
}

// truthy mirrors JavaScript truthiness for JSON values.
func truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case float64:
		return t != 0
	case int:
		return t != 0
	case string:
		return t != ""
	}
	return true
}

// ltNumber mirrors `a < b` with a number literal.
func ltNumber(v any, n float64) bool {
	switch t := v.(type) {
	case float64:
		return t < n
	case int:
		return float64(t) < n
	case string:
		f, err := strconv.ParseFloat(t, 64)
		return err == nil && f < n
	}
	return false
}

// nullish mirrors `x ??= 0`.
func defaultZero(r *dal.Record, k string) {
	if get(r, k) == nil {
		r.Set(k, 0)
	}
}

func orZero(v any) any {
	if v == nil {
		return 0
	}
	return v
}

func requireItem(c ActionContext, missingRef string) (*dal.Record, any, error) {
	refID := get(details(c.Action), "reference_id")
	if refID == nil {
		return nil, nil, invalid(missingRef)
	}
	existing, _ := findItem2(c.CurrentOrder, refID)
	if existing == nil {
		return nil, refID, invalid(fmt.Sprintf(`Item ID "%v" not found.`, refID))
	}
	return existing, refID, nil
}

func quantity(c ActionContext) any { return get(details(c.Action), "quantity") }

// adjustDetail adds (sign 1) or subtracts (sign -1) the action quantity to
// a detail quantity of the referenced item.
func adjustDetail(field string, sign int) func(c ActionContext) (any, error) {
	return func(c ActionContext) (any, error) {
		existing, _ := findItem2(c.CurrentOrder, get(details(c.Action), "reference_id"))
		d := detailOf(existing)
		defaultZero(d, field)
		if sign > 0 {
			d.Set(field, totals.Add(get(d, field), quantity(c)))
		} else {
			d.Set(field, totals.Sub(get(d, field), quantity(c)))
		}
		setActionReference(existing, c.Action, c.Options)
		return nil, nil
	}
}

func init() {
	RegisterActionType(CancelItemFulfillment, ActionTypeDefinition{
		Operation: adjustDetail("fulfilled_quantity", -1),
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to cancel item fulfillment %v is required.", refID))
			}
			if ltNumber(quantity(c), 1) {
				return invalid(fmt.Sprintf("Quantity to cancel item %v must be greater than 0.", refID))
			}
			if totals.Gt(quantity(c), get(detailOf(existing), "fulfilled_quantity")) {
				return invalid(fmt.Sprintf("Cannot cancel more items than what was fulfilled for item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(CancelReturnItem, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			existing, _ := findItem2(c.CurrentOrder, get(details(c.Action), "reference_id"))
			if _, err := adjustDetail("return_requested_quantity", -1)(c); err != nil {
				return nil, err
			}
			return totals.Mult(get(existing, "unit_price"), quantity(c)), nil
		},
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Details reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to cancel return of item %v is required.", refID))
			}
			if totals.Gt(quantity(c), get(detailOf(existing), "return_requested_quantity")) {
				return invalid(fmt.Sprintf("Cannot cancel more items than what was requested to return for item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(UpdateOrderProperties, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			setActionReference(c.CurrentOrder, c.Action, c.Options)
			return nil, nil
		},
		Validate: func(c ActionContext) error { return nil },
	})

	RegisterActionType(CreditLineAdd, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			creditLines := records(get(c.CurrentOrder, "credit_lines"))
			for _, cl := range creditLines {
				if cl.Has("id") && get(cl, "id") == get(c.Action, "reference_id") {
					return nil, nil
				}
			}
			n := dal.NewRecord()
			n.Set("order_id", get(c.CurrentOrder, "id"))
			n.Set("amount", totals.Convert(get(c.Action, "amount")))
			n.Set("reference", get(c.Action, "reference"))
			n.Set("reference_id", get(c.Action, "reference_id"))
			creditLines = append(creditLines, n)
			setActionReference(n, c.Action, c.Options)
			c.CurrentOrder.Set("credit_lines", toAny(creditLines))
			return nil, nil
		},
		Validate: func(c ActionContext) error {
			if get(c.Action, "amount") == nil {
				return invalid("Amount is required.")
			}
			return nil
		},
	})

	RegisterActionType(DeliverItem, ActionTypeDefinition{
		Operation: adjustDetail("delivered_quantity", 1),
		Validate: func(c ActionContext) error {
			item, refID, err := requireItem(c, "Reference ID is required.")
			if err != nil {
				return err
			}
			d := detailOf(item)
			newDelivered := orZero(quantity(c))
			newTotal := totals.Sum(get(d, "delivered_quantity"), newDelivered)
			if totals.Lte(newDelivered, 0) {
				return invalid(fmt.Sprintf("Quantity of item %v must be greater than 0.", refID))
			}
			if totals.Gt(newTotal, get(d, "fulfilled_quantity")) {
				return invalid(fmt.Sprintf("Cannot deliver more items than what was fulfilled for item %v.", refID))
			}
			if totals.Gt(newTotal, get(item, "quantity")) {
				return invalid(fmt.Sprintf("Cannot deliver more items than what was ordered for item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(FulfillItem, ActionTypeDefinition{
		Operation: adjustDetail("fulfilled_quantity", 1),
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to fulfill of item %v is required.", refID))
			}
			if ltNumber(quantity(c), 1) {
				return invalid(fmt.Sprintf("Quantity of item %v must be greater than 0.", refID))
			}
			notFulfilled := totals.Sub(get(existing, "quantity"), get(detailOf(existing), "fulfilled_quantity"))
			if totals.Gt(quantity(c), notFulfilled) {
				return invalid(fmt.Sprintf("Cannot fulfill more items than what was ordered for item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(ItemAdd, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			d := details(c.Action)
			existing, _ := findItem2(c.CurrentOrder, get(d, "reference_id"))
			if existing != nil {
				det := detailOf(existing)
				defaultZero(det, "quantity")
				existing.Set("quantity", totals.Add(get(existing, "quantity"), get(d, "quantity")))
				det.Set("quantity", totals.Add(get(det, "quantity"), get(d, "quantity")))
				if m, ok := d.Get("metadata"); ok {
					if m == nil {
						existing.Set("metadata", nil)
					} else if mr, ok := m.(*dal.Record); ok {
						existing.Set("metadata", utilsmodulessdk.MergeMetadata(orEmptyRecord(get(existing, "metadata")), mr))
					}
				}
			} else {
				existing = dal.NewRecord()
				existing.Set("id", get(d, "reference_id"))
				existing.Set("order_id", get(c.CurrentOrder, "id"))
				existing.Set("return_id", get(c.Action, "return_id"))
				existing.Set("claim_id", get(c.Action, "claim_id"))
				existing.Set("exchange_id", get(c.Action, "exchange_id"))
				existing.Set("unit_price", get(d, "unit_price"))
				if v, ok := d.Get("compare_at_unit_price"); ok {
					existing.Set("compare_at_unit_price", v)
				}
				existing.Set("quantity", get(d, "quantity"))
				if m, ok := d.Get("metadata"); ok {
					existing.Set("metadata", m)
				}
				items := records(get(c.CurrentOrder, "items"))
				c.CurrentOrder.Set("items", append(toAny(items), existing))
			}
			setActionReference(existing, c.Action, c.Options)
			return totals.Mult(get(d, "unit_price"), get(d, "quantity")), nil
		},
		Validate: func(c ActionContext) error {
			d := details(c.Action)
			refID := get(d, "reference_id")
			if get(c.Action, "amount") == nil && get(d, "unit_price") == nil {
				return invalid(fmt.Sprintf("Unit price of item %v is required if no action.amount is provided.", jsString(refID)))
			}
			if !truthy(get(d, "quantity")) {
				return invalid(fmt.Sprintf("Quantity of item %v is required.", jsString(refID)))
			}
			if totals.Lt(get(d, "quantity"), 1) {
				return invalid(fmt.Sprintf("Quantity of item %v must be greater than 0.", jsString(refID)))
			}
			return nil
		},
	})

	RegisterActionType(ItemAdjustmentsReplace, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			d := details(c.Action)
			existing, _ := findItem2(c.CurrentOrder, get(d, "reference_id"))
			if existing == nil {
				return nil, nil
			}
			adj := get(d, "adjustments")
			if adj == nil {
				adj = []any{}
			}
			existing.Set("adjustments", adj)
			setActionReference(existing, c.Action, c.Options)
			return nil, nil
		},
		Validate: func(c ActionContext) error {
			d := details(c.Action)
			if !truthy(get(d, "adjustments")) {
				return invalid(fmt.Sprintf("Adjustments of item %v must exist.", jsString(get(d, "reference_id"))))
			}
			return nil
		},
	})

	RegisterActionType(ItemRemove, ActionTypeDefinition{
		IsDeduction: true,
		Operation: func(c ActionContext) (any, error) {
			d := details(c.Action)
			existing, _ := findItem2(c.CurrentOrder, get(d, "reference_id"))
			det := detailOf(existing)
			defaultZero(det, "quantity")
			existing.Set("quantity", totals.Sub(get(existing, "quantity"), get(d, "quantity")))
			det.Set("quantity", totals.Sub(get(det, "quantity"), get(d, "quantity")))
			setActionReference(existing, c.Action, c.Options)
			return totals.Mult(get(existing, "unit_price"), get(d, "quantity")), nil
		},
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity of item %v is required.", refID))
			}
			if totals.Lt(quantity(c), 1) {
				return invalid(fmt.Sprintf("Quantity of item %v must be greater than 0.", refID))
			}
			notFulfilled := totals.Sub(get(existing, "quantity"), get(detailOf(existing), "fulfilled_quantity"))
			if totals.Gt(quantity(c), notFulfilled) {
				return invalid(fmt.Sprintf("Cannot remove fulfilled item: Item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(ItemUpdate, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			d := details(c.Action)
			existing, _ := findItem2(c.CurrentOrder, get(d, "reference_id"))
			det := detailOf(existing)
			originalQuantity := totals.Convert(coalesceAny(get(det, "quantity"), get(existing, "quantity")))
			originalUnitPrice := totals.Convert(coalesceAny(get(det, "unit_price"), get(existing, "unit_price")))
			currentQuantity := totals.Convert(get(d, "quantity"))
			quantityDiff := totals.Sub(currentQuantity, originalQuantity)
			existing.Set("quantity", currentQuantity)
			det.Set("quantity", currentQuantity)
			if m, ok := d.Get("metadata"); ok {
				if m == nil {
					det.Set("metadata", nil)
					existing.Set("metadata", nil)
				} else if mr, ok := m.(*dal.Record); ok {
					det.Set("metadata", utilsmodulessdk.MergeMetadata(orEmptyRecord(coalesceAny(get(det, "metadata"), get(existing, "metadata"))), mr))
					existing.Set("metadata", utilsmodulessdk.MergeMetadata(orEmptyRecord(get(existing, "metadata")), mr))
				}
			}
			if truthy(get(d, "adjustments")) {
				existing.Set("adjustments", get(d, "adjustments"))
			}
			if get(d, "unit_price") != nil {
				currentUnitPrice := totals.Convert(get(d, "unit_price"))
				originalTotal := totals.Mult(originalUnitPrice, originalQuantity)
				currentTotal := totals.Mult(currentUnitPrice, currentQuantity)
				existing.Set("unit_price", currentUnitPrice)
				det.Set("unit_price", currentUnitPrice)
				setActionReference(existing, c.Action, c.Options)
				return totals.Sub(currentTotal, originalTotal), nil
			}
			setActionReference(existing, c.Action, c.Options)
			return totals.Mult(get(existing, "unit_price"), quantityDiff), nil
		},
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Reference ID is required.")
			if err != nil {
				return err
			}
			d := details(c.Action)
			if get(d, "quantity") == nil {
				return invalid(fmt.Sprintf("Quantity of item %v is required.", refID))
			}
			fulfilled := get(detailOf(existing), "fulfilled_quantity")
			if totals.Lt(get(d, "quantity"), fulfilled) {
				return invalid(fmt.Sprintf("Item %v has already been fulfilled and quantity cannot be lower than %v.", refID, jsString(fulfilled)))
			}
			return nil
		},
	})

	referenceRequired := func(c ActionContext) error {
		if !truthy(get(c.Action, "reference_id")) {
			return invalid("Reference ID is required.")
		}
		return nil
	}
	noop := func(c ActionContext) (any, error) { return nil, nil }
	RegisterActionType(PromotionAdd, ActionTypeDefinition{Operation: noop, Validate: referenceRequired})
	RegisterActionType(PromotionRemove, ActionTypeDefinition{Operation: noop, Validate: referenceRequired})
	RegisterActionType(ShippingUpdate, ActionTypeDefinition{Operation: noop, Validate: referenceRequired})

	receive := func(field string, writeOff bool) func(c ActionContext) (any, error) {
		return func(c ActionContext) (any, error) {
			existing, _ := findItem2(c.CurrentOrder, get(details(c.Action), "reference_id"))
			d := detailOf(existing)
			toReturn := quantity(c)
			defaultZero(d, field)
			defaultZero(d, "return_requested_quantity")
			d.Set(field, totals.Add(get(d, field), toReturn))
			d.Set("return_requested_quantity", totals.Sub(get(d, "return_requested_quantity"), toReturn))
			if writeOff {
				defaultZero(d, "written_off_quantity")
				d.Set("written_off_quantity", totals.Add(get(d, "written_off_quantity"), quantity(c)))
			}
			setActionReference(existing, c.Action, c.Options)
			return totals.Mult(get(existing, "unit_price"), quantity(c)), nil
		}
	}
	RegisterActionType(ReceiveDamagedReturnItem, ActionTypeDefinition{
		IsDeduction: true,
		Operation:   receive("return_dismissed_quantity", true),
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Details reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to return of item %v is required.", refID))
			}
			requested := get(detailOf(existing), "return_requested_quantity")
			if !truthy(requested) {
				requested = 0
			}
			if totals.Gt(quantity(c), requested) {
				return invalid(fmt.Sprintf("Cannot receive more items than what was requested to be returned for item %v.", refID))
			}
			return nil
		},
	})
	RegisterActionType(ReceiveReturnItem, ActionTypeDefinition{
		IsDeduction: true,
		Operation:   receive("return_received_quantity", false),
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Details reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to receive return of item %v is required.", refID))
			}
			requested := get(detailOf(existing), "return_requested_quantity")
			if !truthy(requested) {
				requested = 0
			}
			if totals.Gt(quantity(c), requested) {
				return invalid(fmt.Sprintf("Cannot receive more items than what was requested to be returned for item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(ReinstateItem, ActionTypeDefinition{
		Operation: adjustDetail("written_off_quantity", -1),
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Details reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to reinstate item %v is required.", refID))
			}
			if totals.Lte(quantity(c), 0) {
				return invalid(fmt.Sprintf("Quantity to reinstate item %v must be greater than 0.", refID))
			}
			if totals.Gt(quantity(c), orZero(get(detailOf(existing), "written_off_quantity"))) {
				return invalid("Cannot unclaim more items than what was claimed.")
			}
			return nil
		},
	})

	RegisterActionType(ReturnItem, ActionTypeDefinition{
		IsDeduction: true,
		Operation: func(c ActionContext) (any, error) {
			existing, _ := findItem2(c.CurrentOrder, get(details(c.Action), "reference_id"))
			if _, err := adjustDetail("return_requested_quantity", 1)(c); err != nil {
				return nil, err
			}
			return totals.Mult(get(existing, "unit_price"), quantity(c)), nil
		},
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Details reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to return of item %v is required.", refID))
			}
			d := detailOf(existing)
			available := totals.Sub(orZero(get(d, "fulfilled_quantity")), totals.Add(orZero(get(d, "return_requested_quantity")), orZero(get(d, "return_received_quantity")), orZero(get(d, "return_dismissed_quantity"))))
			if totals.Gt(quantity(c), available) {
				return invalid(fmt.Sprintf("Cannot request to return more items than what was fulfilled for item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(ShipItem, ActionTypeDefinition{
		Operation: adjustDetail("shipped_quantity", 1),
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to ship of item %v is required.", refID))
			}
			if totals.Lt(quantity(c), 1) {
				return invalid(fmt.Sprintf("Quantity of item %v must be greater than 0.", refID))
			}
			d := detailOf(existing)
			notShipped := totals.Sub(get(d, "fulfilled_quantity"), get(d, "shipped_quantity"))
			if totals.Gt(quantity(c), notShipped) {
				return invalid(fmt.Sprintf("Cannot ship more items than what was fulfilled for item %v.", refID))
			}
			return nil
		},
	})

	RegisterActionType(ShippingAdd, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			shipping := shippingList(c.CurrentOrder)
			var existing *dal.Record
			for _, sh := range shipping {
				if get(sh, "id") == get(c.Action, "reference_id") {
					existing = sh
				}
			}
			if existing == nil {
				existing = dal.NewRecord()
				existing.Set("id", get(c.Action, "reference_id"))
				existing.Set("order_id", get(c.CurrentOrder, "id"))
				existing.Set("return_id", get(c.Action, "return_id"))
				existing.Set("claim_id", get(c.Action, "claim_id"))
				existing.Set("exchange_id", get(c.Action, "exchange_id"))
				existing.Set("amount", get(c.Action, "amount"))
				shipping = append(shipping, existing)
			}
			setActionReference(existing, c.Action, c.Options)
			c.CurrentOrder.Set("shipping_methods", toAny(shipping))
			return nil, nil
		},
		Validate: func(c ActionContext) error {
			if !truthy(get(c.Action, "reference_id")) {
				return invalid("Reference ID is required.")
			}
			if get(c.Action, "amount") == nil {
				return invalid("Amount is required.")
			}
			return nil
		},
	})

	RegisterActionType(ShippingAdjustmentsReplace, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			d := details(c.Action)
			var existing *dal.Record
			for _, sm := range records(get(c.CurrentOrder, "shipping_methods")) {
				if get(sm, "id") == get(d, "reference_id") {
					existing = sm
					break
				}
			}
			if existing == nil {
				return nil, nil
			}
			adj := get(d, "adjustments")
			if adj == nil {
				adj = []any{}
			}
			existing.Set("adjustments", adj)
			setActionReference(existing, c.Action, c.Options)
			return nil, nil
		},
		Validate: func(c ActionContext) error {
			d := details(c.Action)
			if !truthy(get(d, "adjustments")) {
				return invalid(fmt.Sprintf("Adjustments of shipping method %v must exist.", jsString(get(d, "reference_id"))))
			}
			return nil
		},
	})

	RegisterActionType(ShippingRemove, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			shipping := shippingList(c.CurrentOrder)
			for i, sm := range shipping {
				if get(sm, "id") == get(c.Action, "reference_id") {
					shipping = append(shipping[:i], shipping[i+1:]...)
					break
				}
			}
			c.CurrentOrder.Set("shipping_methods", toAny(shipping))
			return nil, nil
		},
		Validate: referenceRequired,
	})

	RegisterActionType(TransferCustomer, ActionTypeDefinition{
		Operation: func(c ActionContext) (any, error) {
			c.CurrentOrder.Set("customer_id", get(c.Action, "reference_id"))
			if email := get(details(c.Action), "new_email"); truthy(email) {
				c.CurrentOrder.Set("email", email)
			}
			setActionReference(c.CurrentOrder, c.Action, c.Options)
			return nil, nil
		},
		Validate: func(c ActionContext) error {
			if !truthy(get(c.Action, "reference_id")) {
				return invalid("Reference to customer ID is required")
			}
			return nil
		},
	})

	RegisterActionType(WriteOffItem, ActionTypeDefinition{
		Operation: adjustDetail("written_off_quantity", 1),
		Validate: func(c ActionContext) error {
			existing, refID, err := requireItem(c, "Details reference ID is required.")
			if err != nil {
				return err
			}
			if !truthy(quantity(c)) {
				return invalid(fmt.Sprintf("Quantity to write-off item %v is required.", refID))
			}
			if totals.Lte(quantity(c), 0) {
				return invalid(fmt.Sprintf("Quantity to write-off item %v must be greater than 0.", refID))
			}
			available := totals.Sub(orZero(get(existing, "quantity")), orZero(get(detailOf(existing), "written_off_quantity")))
			if totals.Gt(quantity(c), available) {
				return invalid("Cannot claim more items than what was ordered.")
			}
			return nil
		},
	})
}

func shippingList(order *dal.Record) []*dal.Record {
	v := get(order, "shipping_methods")
	if r, ok := v.(*dal.Record); ok {
		return []*dal.Record{r}
	}
	return records(v)
}

func orEmptyRecord(v any) any {
	if v == nil {
		return dal.NewRecord()
	}
	return v
}

func coalesceAny(vs ...any) any {
	for _, v := range vs {
		if v != nil {
			return v
		}
	}
	return nil
}

// jsString mirrors `${v}` in a template literal.
func jsString(v any) string {
	switch t := v.(type) {
	case nil:
		return "undefined"
	case float64:
		return totals.JSNumberToString(t)
	case *totals.BN:
		return t.Text('f')
	}
	return fmt.Sprint(v)
}
