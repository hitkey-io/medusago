// Package utils mirrors @medusajs/order/src/utils.
package utils

import (
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	carttotals "github.com/hitkey-io/medusago/packages/core/utils/src/totals/cart"
)

var orderItemFields = []string{"quantity", "unit_price", "compare_at_unit_price"}

func get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func isDefined(r *dal.Record, k string) bool {
	if r == nil {
		return false
	}
	_, ok := r.Get(k)
	return ok
}

func records(v any) []*dal.Record {
	switch list := v.(type) {
	case []*dal.Record:
		return list
	case []any:
		out := make([]*dal.Record, 0, len(list))
		for _, x := range list {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

func toAny(list []*dal.Record) []any {
	out := make([]any, len(list))
	for i, r := range list {
		out[i] = r
	}
	return out
}

func copyRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

// coalesce mirrors `a ?? b` on record fields.
func coalesce(a *dal.Record, ka string, b *dal.Record, kb string) any {
	if v := get(a, ka); v != nil {
		return v
	}
	return get(b, kb)
}

// FormatOrderOptions mirrors the options of formatOrder.
type FormatOrderOptions struct {
	// Entity is the class name of the listed entity (Order, Return,
	// OrderClaim, OrderExchange).
	Entity        string
	IncludeTotals bool
}

// FormatOrder mirrors formatOrder: it reshapes the repository rows into the
// order DTO (in place) and decorates the totals.
func FormatOrder(orders []*dal.Record, opts FormatOrderOptions) []*dal.Record {
	isRelatedEntity := opts.Entity != "Order"
	for _, order := range orders {
		mainOrder := order
		if isRelatedEntity {
			o, _ := get(order, "order").(*dal.Record)
			if o == nil {
				continue
			}
			mainOrder = o
		}

		if items, ok := mainOrder.Get("items"); ok && items != nil {
			list := records(items)
			formatted := make([]any, 0, len(list))
			for _, orderItem := range list {
				detailIn, _ := get(orderItem, "detail").(*dal.Record)
				if detailIn != nil && get(detailIn, "fulfilled_quantity") != nil {
					formatted = append(formatted, orderItem)
					continue
				}
				detail := copyRecord(orderItem)
				detail.Delete("order")
				detail.Delete("item")
				item, _ := get(orderItem, "item").(*dal.Record)
				out := copyRecord(item)
				out.Set("quantity", get(detail, "quantity"))
				out.Set("raw_quantity", get(detail, "raw_quantity"))
				out.Set("unit_price", coalesce(detail, "unit_price", item, "unit_price"))
				out.Set("raw_unit_price", coalesce(detail, "raw_unit_price", item, "raw_unit_price"))
				out.Set("compare_at_unit_price", coalesce(detail, "compare_at_unit_price", item, "compare_at_unit_price"))
				out.Set("raw_compare_at_unit_price", coalesce(detail, "raw_compare_at_unit_price", item, "raw_compare_at_unit_price"))
				out.Set("metadata", coalesce(detail, "metadata", item, "metadata"))
				out.Set("line_item_metadata", get(item, "metadata"))
				out.Set("detail", detail)
				formatted = append(formatted, out)
			}
			mainOrder.Set("items", formatted)
		}

		if isRelatedEntity {
			if ret, ok := get(order, "return").(*dal.Record); ok && ret != nil {
				formatOrderReturn(ret, mainOrder)
			}
			switch opts.Entity {
			case "OrderClaim":
				formatClaim(order)
			case "OrderExchange":
				formatExchange(order)
			case "Return":
				formatReturn(order)
			}
		}

		if sms, ok := order.Get("shipping_methods"); ok && sms != nil {
			list := records(sms)
			formatted := make([]any, 0, len(list))
			for _, shippingMethod := range list {
				if get(shippingMethod, "detail") != nil {
					formatted = append(formatted, shippingMethod)
					continue
				}
				sm := copyRecord(asRecord(get(shippingMethod, "shipping_method")))
				shippingMethod.Delete("shipping_method")
				cleanNestedRelations(shippingMethod)
				sm.Set("order_id", get(shippingMethod, "order_id"))
				sm.Set("detail", copyRecord(shippingMethod))
				formatted = append(formatted, sm)
			}
			order.Set("shipping_methods", formatted)
		}

		if summary, ok := mainOrder.Get("summary"); ok && summary != nil {
			list := records(summary)
			var totalsValue any
			if len(list) > 0 {
				totalsValue = get(list[0], "totals")
			}
			mainOrder.Set("summary", totalsValue)
		}

		if opts.IncludeTotals {
			carttotals.DecorateCartTotals(order, false)
		}
		totals.CreateRawPropertiesFromBigNumber(order, func() totals.Object { return dal.NewRecord() })
	}
	return orders
}

func asRecord(v any) *dal.Record {
	r, _ := v.(*dal.Record)
	return r
}

func cleanNestedRelations(obj *dal.Record) {
	if obj == nil {
		return
	}
	obj.Delete("order")
	obj.Delete("return")
	obj.Delete("claim")
	obj.Delete("exchange")
}

func findItem(order *dal.Record, id any) *dal.Record {
	for _, item := range records(get(order, "items")) {
		if get(item, "id") == id {
			return item
		}
	}
	return nil
}

// setItemDetail mirrors orderItem.detail = item?.detail: an undefined detail
// (no matching order item) drops the key.
func setItemDetail(orderItem, item *dal.Record) {
	if item != nil {
		if d, ok := item.Get("detail"); ok {
			orderItem.Set("detail", d)
			return
		}
	}
	orderItem.Delete("detail")
}

func formatOrderReturn(orderReturn, mainOrder *dal.Record) {
	for _, orderItem := range records(get(orderReturn, "items")) {
		item := findItem(mainOrder, get(orderItem, "item_id"))
		setItemDetail(orderItem, item)
	}
}

func formatClaim(claim *dal.Record) {
	order := asRecord(get(claim, "order"))
	// claim_items and additional_items load the same OrderClaimItem entities:
	// a flag hydrated through one relation is visible through the other
	additional := map[any]bool{}
	for _, rel := range []string{"claim_items", "additional_items"} {
		for _, it := range records(get(claim, rel)) {
			if b, ok := get(it, "is_additional_item").(bool); ok {
				additional[get(it, "id")] = b
			}
		}
	}
	isAdditional := func(it *dal.Record) bool { return additional[get(it, "id")] }
	if v, ok := claim.Get("additional_items"); ok && v != nil {
		var kept []*dal.Record
		for _, it := range records(v) {
			if isAdditional(it) {
				kept = append(kept, it)
			}
		}
		for _, orderItem := range kept {
			item := findItem(order, get(orderItem, "item_id"))
			cleanNestedRelations(item)
			setItemDetail(orderItem, item)
		}
		claim.Set("additional_items", toAny(kept))
	}
	v, ok := claim.Get("claim_items")
	if !ok || v == nil {
		return
	}
	var kept []*dal.Record
	for _, it := range records(v) {
		if !isAdditional(it) {
			kept = append(kept, it)
		}
	}
	for _, orderItem := range kept {
		item := findItem(order, get(orderItem, "item_id"))
		cleanNestedRelations(item)
		setItemDetail(orderItem, item)
	}
	claim.Set("claim_items", toAny(kept))
}

func formatExchange(exchange *dal.Record) {
	order := asRecord(get(exchange, "order"))
	for _, orderItem := range records(get(exchange, "additional_items")) {
		item := findItem(order, get(orderItem, "item_id"))
		cleanNestedRelations(item)
		setItemDetail(orderItem, item)
	}
}

func formatReturn(returnOrder *dal.Record) {
	order := asRecord(get(returnOrder, "order"))
	for _, orderItem := range records(get(returnOrder, "items")) {
		item := findItem(order, get(orderItem, "item_id"))
		cleanNestedRelations(item)
		setItemDetail(orderItem, item)
	}
}

// deduplicate mirrors deduplicate (first occurrence wins).
func deduplicate(list []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, x := range list {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

// jsSort mirrors Array.prototype.sort without a comparator (UTF-16 code
// unit order; the strings here are ASCII).
func jsSort(list []string) {
	sort.SliceStable(list, func(i, j int) bool { return list[i] < list[j] })
}

// RepositoryConfig is the part of the find options mapRepositoryToOrderModel
// rewrites: options.fields, options.populate and where.
type RepositoryConfig struct {
	Fields   []string
	Populate []string
	Where    *dal.Record
	// HasFields / HasPopulate report whether the option is defined.
	HasFields, HasPopulate bool
}

// MapRepositoryToOrderModel mirrors mapRepositoryToOrderModel.
func MapRepositoryToOrderModel(conf *RepositoryConfig, isRelatedEntity bool) {
	if isRelatedEntity {
		mapRepositoryToRelatedEntity(conf)
		return
	}
	replace := func(list []string, isFields bool) []string {
		jsSort(list)
		var out []string
		for _, rel := range list {
			switch {
			case rel == "summary" && isFields:
				conf.Populate = append(conf.Populate, "summary")
				conf.HasPopulate = true
				out = append(out, "summary.totals")
			case strings.Contains(rel, "shipping_methods") && !strings.Contains(rel, "shipping_methods.shipping_method"):
				conf.Populate = append(conf.Populate, "shipping_methods.shipping_method")
				conf.HasPopulate = true
				out = append(out, strings.Replace(rel, "shipping_methods", "shipping_methods.shipping_method", 1))
			case strings.Contains(rel, "items.detail"):
				out = append(out, strings.Replace(rel, "items.detail", "items", 1))
			case rel == "items":
				out = append(out, "items.item")
			case isFields && isOrderItemField(rel):
				out = append(out, rel, strings.Replace(rel, "items.", "items.item.", 1))
			case strings.Contains(rel, "items.") && !strings.Contains(rel, "items.item"):
				out = append(out, strings.Replace(rel, "items.", "items.item.", 1))
			default:
				out = append(out, rel)
			}
		}
		return deduplicate(out)
	}
	if conf.HasFields {
		conf.Fields = replace(conf.Fields, true)
	}
	if conf.HasPopulate {
		conf.Populate = replace(conf.Populate, false)
	}

	if conf.Where != nil {
		if items, ok := conf.Where.Get("items"); ok {
			original := copyRecord(asRecord(items))
			inner := asRecord(items)
			if inner == nil {
				return
			}
			detail := asRecord(get(original, "detail"))
			if detail != nil {
				inner.Delete("detail")
			}
			wrapped := dal.NewRecord()
			wrapped.Set("item", inner)
			for _, k := range orderItemFields {
				if v := get(original, k); v != nil {
					wrapped.Set(k, v)
					inner.Delete(k)
				}
			}
			if detail != nil {
				merged := copyRecord(detail)
				for _, k := range wrapped.Keys() {
					v, _ := wrapped.Get(k)
					merged.Set(k, v)
				}
				wrapped = merged
			}
			conf.Where.Set("items", wrapped)
		}
	}
}

func isOrderItemField(rel string) bool {
	for _, f := range orderItemFields {
		if rel == "items."+f {
			return true
		}
	}
	return false
}

func mapRepositoryToRelatedEntity(conf *RepositoryConfig) {
	replace := func(list []string) []string {
		jsSort(list)
		out := make([]string, 0, len(list))
		for _, rel := range list {
			if strings.Contains(rel, "shipping_methods") && !strings.Contains(rel, "shipping_methods.shipping_method") {
				conf.Populate = append(conf.Populate, "shipping_methods.shipping_method")
				conf.HasPopulate = true
				out = append(out, strings.Replace(rel, "shipping_methods", "shipping_methods.shipping_method", 1))
				continue
			}
			out = append(out, rel)
		}
		return deduplicate(out)
	}
	if conf.HasFields {
		conf.Fields = replace(conf.Fields)
	}
	if conf.HasPopulate {
		conf.Populate = replace(conf.Populate)
	}
}
