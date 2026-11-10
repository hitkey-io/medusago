// Package creditlines mirrors totals/credit-lines.
package creditlines

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/defaults"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// CurrencyEpsilon mirrors getEpsilonFromDecimalPrecision(defaultCurrencies[code]?.decimal_digits).
func CurrencyEpsilon(code string) *totals.BN {
	upper := strings.ToUpper(code)
	for _, c := range defaults.DefaultCurrencies {
		if c.Code == upper {
			return totals.GetEpsilonFromDecimalPrecision(c.DecimalDigits)
		}
	}
	return totals.GetEpsilonFromDecimalPrecision(-1)
}

// CalculateCreditLinesTotal mirrors calculateCreditLinesTotal (taxRate nil
// = undefined).
func CalculateCreditLinesTotal(creditLines []totals.Object, includesTax bool, taxRate any, currencyCode string) (total, subtotal, taxTotal *totals.BN) {
	subtotal = totals.Convert(0)
	total = totals.Convert(0)
	taxTotal = totals.Convert(0)
	for _, cl := range creditLines {
		a, ok := cl.Get("amount")
		if !ok {
			continue
		}
		amount := totals.Convert(a)
		subtotal = totals.Add(subtotal, amount)
		if taxRate != nil {
			clSub := amount
			if includesTax {
				clSub = totals.Div(amount, totals.Add(1, taxRate))
			}
			clTax := totals.Mult(clSub, taxRate)
			clTotal := totals.Add(clSub, clTax)
			cl.Set("subtotal", totals.NewBN(clSub))
			cl.Set("total", totals.NewBN(clTotal))
			total = totals.Add(total, clTotal)
			taxTotal = totals.Add(taxTotal, clTax)
		} else {
			cl.Set("subtotal", totals.NewBN(amount))
			total = totals.Add(total, amount)
		}
	}
	if totals.Lte(totals.Abs(total), CurrencyEpsilon(currencyCode)) {
		return totals.Convert(0), totals.Convert(0), totals.Convert(0)
	}
	return total, subtotal, taxTotal
}
