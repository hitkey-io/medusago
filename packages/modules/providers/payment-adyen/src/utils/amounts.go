// Package utils mirrors @medusajs/payment-adyen/src/utils.
package utils

import (
	"math"
	"strconv"
	"strings"

	"github.com/cockroachdb/apd/v3"

	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// Adyen's own exponents, which differ from ISO 4217 for some currencies.
var zeroDecimal = map[string]bool{
	"CVE": true, "DJF": true, "GNF": true, "IDR": true, "JPY": true, "KMF": true, "KRW": true, "PYG": true,
	"RWF": true, "UGX": true, "VND": true, "VUV": true, "XAF": true, "XOF": true, "XPF": true,
}

var threeDecimal = map[string]bool{
	"BHD": true, "IQD": true, "JOD": true, "KWD": true, "LYD": true, "OMR": true, "TND": true,
}

// GetCurrencyExponent mirrors getCurrencyExponent.
func GetCurrencyExponent(currency string) int {
	code := strings.ToUpper(currency)
	if zeroDecimal[code] {
		return 0
	}
	if threeDecimal[code] {
		return 3
	}
	return 2
}

func toFloat(d *totals.BN) float64 {
	f, _ := strconv.ParseFloat(d.String(), 64)
	return f
}

// ToMinorUnits mirrors toMinorUnits (rounded half up to an integer).
func ToMinorUnits(amount any, currency string) float64 {
	multiplier := math.Pow(10, float64(GetCurrencyExponent(currency)))
	v := totals.Mult(amount, multiplier)
	out := new(apd.Decimal)
	ctx := apd.BaseContext.WithPrecision(1000)
	ctx.Rounding = apd.RoundHalfUp
	_, _ = ctx.Quantize(out, v, 0)
	return toFloat(out)
}

// FromMinorUnits mirrors fromMinorUnits.
func FromMinorUnits(amount any, currency string) float64 {
	multiplier := math.Pow(10, float64(GetCurrencyExponent(currency)))
	return toFloat(totals.Div(amount, multiplier))
}
