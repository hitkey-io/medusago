// Package lineitem mirrors totals/line-item.
package lineitem

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals/adjustment"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals/tax"
)

// Context mirrors GetLineItemsTotalsContext; ExtraQuantityFields is an
// ordered list of [field, totalField] pairs.
type Context struct {
	IncludeTax          bool
	ExtraQuantityFields [][2]string
}

// Totals is an ordered set of computed totals (key order of the JS object).
type Totals struct {
	Keys   []string
	Values map[string]any
}

func (t *Totals) set(k string, v any) { t.setValue(k, totals.NewBN(v)) }

// setValue stores a value as is (item.quantity, item.unit_price).
func (t *Totals) setValue(k string, v any) {
	if t.Values == nil {
		t.Values = map[string]any{}
	}
	if _, ok := t.Values[k]; !ok {
		t.Keys = append(t.Keys, k)
	}
	t.Values[k] = v
}

// Get returns a computed total.
func (t *Totals) Get(k string) any { return t.Values[k] }

// Assign mirrors Object.assign(item, totals).
func (t *Totals) Assign(o totals.Object) {
	for _, k := range t.Keys {
		o.Set(k, t.Values[k])
	}
}

func field(o totals.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

func fieldOr0(o totals.Object, k string) any {
	if o == nil {
		return 0
	}
	v, _ := o.Get(k)
	if v == nil {
		return 0
	}
	return v
}

// GetLineItemsTotals mirrors getLineItemsTotals (keyed by id, or by index
// when the item has no id).
func GetLineItemsTotals(items []totals.Object, ctx Context) map[any]*Totals {
	out := map[any]*Totals{}
	for i, item := range items {
		inc, _ := item.Get("is_tax_inclusive")
		t := GetLineItemTotals(item, Context{IncludeTax: ctx.IncludeTax || totals.Truthy(inc), ExtraQuantityFields: ctx.ExtraQuantityFields})
		if id, ok := item.Get("id"); ok && id != nil {
			out[id] = t
		} else {
			out[i] = t
		}
	}
	return out
}

func isTaxInclusive(item totals.Object, includeTax bool) bool {
	if v, ok := item.Get("is_tax_inclusive"); ok && v != nil {
		return totals.Truthy(v)
	}
	return includeTax
}

func setRefundableTotal(item totals.Object, discountsTotal, discountsSubtotal any, t *Totals) {
	detail := totals.AsObject(field(item, "detail"))
	returned := totals.Sum(fieldOr0(detail, "return_requested_quantity"), fieldOr0(detail, "return_received_quantity"), fieldOr0(detail, "return_dismissed_quantity"))
	quantity := field(item, "quantity")
	current := totals.Sub(quantity, returned)
	inclusive := totals.Truthy(field(item, "is_tax_inclusive"))
	discount := discountsSubtotal
	if inclusive {
		discount = discountsTotal
	}
	perUnit := totals.Div(discount, quantity)
	refundableSub := totals.Sub(totals.Mult(current, field(item, "unit_price")), totals.Mult(current, perUnit))
	taxTotal := tax.CalculateTaxTotal(inclusive, totals.Objects(field(item, "tax_lines")), refundableSub, "")
	refundable := totals.Add(refundableSub, taxTotal)
	if totals.Eq(current, 0) {
		t.set("refundable_total_per_unit", 0)
	} else {
		t.set("refundable_total_per_unit", totals.Div(refundable, current))
	}
	t.set("refundable_total", refundable)
}

// GetLineItemTotals mirrors getLineItemTotals.
func GetLineItemTotals(item totals.Object, ctx Context) *Totals {
	inclusive := isTaxInclusive(item, ctx.IncludeTax)
	taxLines := totals.Objects(field(item, "tax_lines"))
	rates := make([]any, len(taxLines))
	for i, l := range taxLines {
		rates[i] = field(l, "rate")
	}
	sumTaxRate := totals.Div(totals.Sum(rates...), 100)
	unitPrice := field(item, "unit_price")
	quantity := field(item, "quantity")
	totalItemPrice := totals.Mult(unitPrice, quantity)
	subtotal := totalItemPrice
	if inclusive {
		subtotal = totals.Div(totalItemPrice, totals.Add(1, sumTaxRate))
	}
	adj := adjustment.CalculateAdjustmentTotal(quantity, totals.Objects(field(item, "adjustments")), sumTaxRate)

	detail := totals.AsObject(field(item, "detail"))
	returned := totals.Sum(fieldOr0(detail, "return_received_quantity"), fieldOr0(detail, "return_dismissed_quantity"))
	current := totals.Sub(quantity, returned)
	currentTotalItemPrice := totals.Mult(unitPrice, current)
	currentSubtotal := currentTotalItemPrice
	if inclusive {
		currentSubtotal = totals.Div(currentTotalItemPrice, totals.Add(1, sumTaxRate))
	}
	currentDiscountsSubtotal := totals.Mult(adj.AdjustmentSubtotalPerItem, current)
	taxTotal := tax.CalculateTaxTotal(false, taxLines, totals.Sub(currentSubtotal, currentDiscountsSubtotal), "total")
	originalTaxTotal := tax.CalculateTaxTotal(false, taxLines, currentSubtotal, "subtotal")
	fullDiscountedTaxable := totals.Sub(subtotal, adj.AdjustmentsSubtotal)
	taxTotalFull := tax.CalculateTaxTotal(false, taxLines, fullDiscountedTaxable, "")
	fullNetTotal := totals.Sum(fullDiscountedTaxable, taxTotalFull)

	t := &Totals{}
	t.setValue("quantity", quantity)
	t.setValue("unit_price", unitPrice)
	t.set("subtotal", currentSubtotal)
	t.set("total", totals.Sum(totals.Sub(currentSubtotal, currentDiscountsSubtotal), taxTotal))
	originalTotal := totals.Add(currentSubtotal, originalTaxTotal)
	if inclusive {
		originalTotal = currentTotalItemPrice
	}
	t.set("original_subtotal", totals.Sub(originalTotal, originalTaxTotal))
	t.set("original_total", originalTotal)
	t.set("discount_subtotal", currentDiscountsSubtotal)
	t.set("discount_tax_total", totals.Sub(originalTaxTotal, taxTotal))
	t.set("discount_total", totals.Add(currentDiscountsSubtotal, totals.Sub(originalTaxTotal, taxTotal)))
	t.set("tax_total", taxTotal)
	t.set("original_tax_total", originalTaxTotal)

	if detail != nil {
		_, a := detail.Get("return_requested_quantity")
		_, b := detail.Get("return_received_quantity")
		_, c := detail.Get("return_dismissed_quantity")
		if a || b || c {
			setRefundableTotal(item, adj.AdjustmentsTotal, adj.AdjustmentsSubtotal, t)
		}
	}

	var div any = quantity
	if totals.Eq(quantity, 0) {
		div = 1
	}
	totalPerUnit := totals.Div(fullNetTotal, div)
	for _, f := range ctx.ExtraQuantityFields {
		target, ok := totals.PickValueFromObject(f[0], item)
		if !ok {
			continue
		}
		t.set(f[1], totals.Mult(totalPerUnit, target))
	}
	return t
}

// NewTotals returns an empty Totals.
func NewTotals() *Totals { return &Totals{} }

// Set stores new BigNumber(v).
func (t *Totals) Set(k string, v any) { t.set(k, v) }
