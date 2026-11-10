// Package tax mirrors totals/tax.
package tax

import "github.com/hitkey-io/medusago/packages/core/utils/src/totals"

// CalculateTaxTotal mirrors calculateTaxTotal; setTotalField ("" = none)
// stores each line's tax amount on the tax line.
func CalculateTaxTotal(isTaxInclusive bool, taxLines []totals.Object, taxableAmount any, setTotalField string) *totals.BN {
	taxTotal := totals.Convert(0)
	if isTaxInclusive {
		return taxTotal
	}
	for _, line := range taxLines {
		rate, _ := line.Get("rate")
		r := totals.Div(rate, 100)
		amount := totals.Mult(taxableAmount, r)
		if setTotalField != "" {
			line.Set(setTotalField, totals.NewBN(amount))
		}
		taxTotal = totals.Add(taxTotal, amount)
	}
	return taxTotal
}
