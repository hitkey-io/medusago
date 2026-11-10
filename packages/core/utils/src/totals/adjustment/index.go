// Package adjustment mirrors totals/adjustment.
package adjustment

import "github.com/hitkey-io/medusago/packages/core/utils/src/totals"

// Result is the return value of calculateAdjustmentTotal.
type Result struct {
	AdjustmentsTotal, AdjustmentsSubtotal, AdjustmentsTaxTotal                *totals.BN
	AdjustmentPerItem, AdjustmentSubtotalPerItem, AdjustmentTaxTotalPerItem *totals.BN
}

// CalculateAdjustmentTotal mirrors calculateAdjustmentTotal (quantity nil
// = no item; taxRate nil = undefined).
func CalculateAdjustmentTotal(quantity any, adjustments []totals.Object, taxRate any) Result {
	subtotal := totals.Convert(0)
	total := totals.Convert(0)
	taxTotal := totals.Convert(0)
	for _, adj := range adjustments {
		amount, ok := adj.Get("amount")
		if !ok {
			continue
		}
		inclusive, _ := adj.Get("is_tax_inclusive")
		var adjSubtotal any = amount
		if taxRate != nil && totals.Truthy(inclusive) {
			adjSubtotal = totals.Div(amount, totals.Add(1, taxRate))
		}
		var adjTax any = 0
		if taxRate != nil {
			adjTax = totals.Mult(adjSubtotal, taxRate)
		}
		adjTotal := totals.Add(adjSubtotal, adjTax)
		subtotal = totals.Add(subtotal, adjSubtotal)
		taxTotal = totals.Add(taxTotal, adjTax)
		total = totals.Add(total, adjTotal)
		adj.Set("subtotal", totals.NewBN(totals.Convert(adjSubtotal)))
		adj.Set("total", totals.NewBN(adjTotal))
	}
	var q any = quantity
	if quantity == nil || !totals.Truthy(quantity) {
		q = totals.Convert(1)
	}
	r := Result{AdjustmentsTotal: total, AdjustmentsSubtotal: subtotal, AdjustmentsTaxTotal: taxTotal,
		AdjustmentPerItem: totals.Convert(0), AdjustmentSubtotalPerItem: totals.Convert(0), AdjustmentTaxTotalPerItem: totals.Convert(0)}
	if !totals.Eq(q, 0) {
		r.AdjustmentPerItem = totals.Div(total, q)
		r.AdjustmentSubtotalPerItem = totals.Div(subtotal, q)
		r.AdjustmentTaxTotalPerItem = totals.Div(taxTotal, q)
	}
	return r
}
