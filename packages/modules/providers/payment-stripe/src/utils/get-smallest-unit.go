// Package utils mirrors @medusajs/payment-stripe/src/utils.
package utils

import (
	"math"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

var currencyMultipliers = []struct {
	power int
	codes []string
}{
	{0, []string{"BIF", "CLP", "DJF", "GNF", "JPY", "KMF", "KRW", "MGA", "PYG", "RWF", "UGX", "VND", "VUV", "XAF", "XOF", "XPF"}},
	{3, []string{"BHD", "IQD", "JOD", "KWD", "OMR", "TND"}},
}

func getCurrencyMultiplier(currency string) float64 {
	currency = strings.ToUpper(currency)
	power := 2
	for _, m := range currencyMultipliers {
		for _, c := range m.codes {
			if c == currency {
				power = m.power
			}
		}
	}
	return math.Pow(10, float64(power))
}

// numeric mirrors new BigNumber(v).numeric (the JS number of the value).
func numeric(v *totals.BN) float64 {
	f, _ := strconv.ParseFloat(v.String(), 64)
	return f
}

// jsRound mirrors Math.round.
func jsRound(f float64) float64 {
	return math.Floor(f + 0.5)
}

// GetSmallestUnit mirrors getSmallestUnit: the amount in the smallest unit
// of the currency (rounded up to the nearest ten for 3-decimal currencies).
func GetSmallestUnit(amount any, currency string) int64 {
	multiplier := getCurrencyMultiplier(currency)
	amount_ := jsRound(numeric(totals.Mult(amount, multiplier))) / multiplier
	smallest := numeric(totals.Mult(amount_, multiplier))
	if multiplier == 1e3 {
		smallest = math.Ceil(smallest/10) * 10
	}
	s := strconv.FormatFloat(smallest, 'f', -1, 64)
	s, _, _ = strings.Cut(s, ".")
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// GetAmountFromSmallestUnit mirrors getAmountFromSmallestUnit.
func GetAmountFromSmallestUnit(amount any, currency string) float64 {
	multiplier := getCurrencyMultiplier(currency)
	return numeric(totals.Div(amount, multiplier))
}
