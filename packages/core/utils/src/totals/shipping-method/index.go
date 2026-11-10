// Package shippingmethod mirrors totals/shipping-method.
package shippingmethod

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals/adjustment"
	lineitem "github.com/hitkey-io/medusago/packages/core/utils/src/totals/line-item"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals/tax"
)

func field(o totals.Object, k string) any {
	v, _ := o.Get(k)
	return v
}

// GetShippingMethodsTotals mirrors getShippingMethodsTotals.
func GetShippingMethodsTotals(methods []totals.Object, includeTax bool) map[any]*lineitem.Totals {
	out := map[any]*lineitem.Totals{}
	for i, m := range methods {
		t := GetShippingMethodTotals(m, includeTax)
		if id, ok := m.Get("id"); ok && id != nil {
			out[id] = t
		} else {
			out[i] = t
		}
	}
	return out
}

// GetShippingMethodTotals mirrors getShippingMethodTotals.
func GetShippingMethodTotals(m totals.Object, includeTax bool) *lineitem.Totals {
	inclusive := includeTax
	if v, ok := m.Get("is_tax_inclusive"); ok && v != nil {
		inclusive = totals.Truthy(v)
	}
	amount := totals.Convert(field(m, "amount"))
	taxLines := totals.Objects(field(m, "tax_lines"))
	rates := make([]any, len(taxLines))
	for i, l := range taxLines {
		rates[i] = field(l, "rate")
	}
	sumTaxRate := totals.Div(totals.Sum(rates...), 100)
	subtotal := amount
	if inclusive {
		subtotal = totals.Div(amount, totals.Add(1, sumTaxRate))
	}
	adj := adjustment.CalculateAdjustmentTotal(nil, totals.Objects(field(m, "adjustments")), sumTaxRate)
	taxTotal := tax.CalculateTaxTotal(false, taxLines, totals.Sub(subtotal, adj.AdjustmentsSubtotal), "total")
	originalTaxTotal := tax.CalculateTaxTotal(false, taxLines, subtotal, "subtotal")
	t := lineitem.NewTotals()
	t.Set("amount", amount)
	t.Set("subtotal", subtotal)
	t.Set("total", totals.Sum(totals.Sub(subtotal, adj.AdjustmentsSubtotal), taxTotal))
	originalTotal := totals.Add(subtotal, originalTaxTotal)
	if inclusive {
		originalTotal = amount
	}
	t.Set("original_subtotal", totals.Sub(originalTotal, originalTaxTotal))
	t.Set("original_total", originalTotal)
	t.Set("discount_total", adj.AdjustmentsTotal)
	t.Set("discount_subtotal", adj.AdjustmentsSubtotal)
	t.Set("discount_tax_total", adj.AdjustmentsTaxTotal)
	t.Set("tax_total", taxTotal)
	t.Set("original_tax_total", originalTaxTotal)
	return t
}
