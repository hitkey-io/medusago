package totals

import (
	"strconv"

	"github.com/cockroachdb/apd/v3"
)

// bnContext mirrors bignumber.js defaults: DECIMAL_PLACES 20 for divisions,
// ROUND_HALF_UP. Additions, subtractions and multiplications are exact.
var bnContext = apd.BaseContext.WithPrecision(1000)

// BN is a bignumber.js value.
type BN = apd.Decimal

// Convert mirrors MathBN.convert for the values found in records: nil,
// numbers, numeric strings, raw values ({ value, precision }) and
// *BigNumber.
func Convert(v any) *BN {
	d := new(apd.Decimal)
	switch t := v.(type) {
	case nil:
		return d
	case *apd.Decimal:
		d.Set(t)
	case *BigNumber:
		d.Set(&t.decimal)
	case BigNumberRawValue:
		_, _, _ = d.SetString(t.Value)
	case interface{ Get(string) (any, bool) }:
		// a raw value record { value, precision }
		if val, ok := t.Get("value"); ok && val != nil {
			return Convert(val)
		}
		_, _, _ = d.SetString("NaN")
	case float64:
		_, _, _ = d.SetString(strconv.FormatFloat(t, 'g', -1, 64))
	case int:
		d.SetInt64(int64(t))
	case int64:
		d.SetInt64(t)
	case string:
		if _, _, err := d.SetString(t); err != nil {
			_, _, _ = d.SetString("NaN")
		}
	default:
		_, _, _ = d.SetString("NaN")
	}
	return d
}

// Add mirrors MathBN.add.
func Add(nums ...any) *BN {
	sum := new(apd.Decimal)
	for _, n := range nums {
		_, _ = bnContext.Add(sum, sum, Convert(n))
	}
	return sum
}

// Sub mirrors MathBN.sub.
func Sub(nums ...any) *BN {
	if len(nums) == 0 {
		return new(apd.Decimal)
	}
	agg := Convert(nums[0])
	for _, n := range nums[1:] {
		_, _ = bnContext.Sub(agg, agg, Convert(n))
	}
	return agg
}

// Mult mirrors MathBN.mult.
func Mult(a, b any) *BN {
	out := new(apd.Decimal)
	_, _ = bnContext.Mul(out, Convert(a), Convert(b))
	return out
}

// Div mirrors MathBN.div (20 decimal places, ROUND_HALF_UP).
func Div(a, b any) *BN {
	out := new(apd.Decimal)
	ctx := bnContext.WithPrecision(1000)
	_, _ = ctx.Quo(out, Convert(a), Convert(b))
	ctx.Rounding = apd.RoundHalfUp
	_, _ = ctx.Quantize(out, out, -20)
	_, _ = out.Reduce(out)
	return out
}

// Abs mirrors MathBN.abs.
func Abs(a any) *BN {
	out := new(apd.Decimal)
	out.Abs(Convert(a))
	return out
}

func cmp(a, b any) int { return Convert(a).Cmp(Convert(b)) }

// Gt mirrors MathBN.gt.
func Gt(a, b any) bool { return cmp(a, b) > 0 }

// Gte mirrors MathBN.gte.
func Gte(a, b any) bool { return cmp(a, b) >= 0 }

// Lt mirrors MathBN.lt.
func Lt(a, b any) bool { return cmp(a, b) < 0 }

// Lte mirrors MathBN.lte.
func Lte(a, b any) bool { return cmp(a, b) <= 0 }

// Eq mirrors MathBN.eq.
func Eq(a, b any) bool { return cmp(a, b) == 0 }

// Min mirrors MathBN.min.
func Min(nums ...any) *BN {
	var m *BN
	for _, n := range nums {
		c := Convert(n)
		if m == nil || c.Cmp(m) < 0 {
			m = c
		}
	}
	if m == nil {
		return new(apd.Decimal)
	}
	return m
}

// Max mirrors MathBN.max.
func Max(nums ...any) *BN {
	var m *BN
	for _, n := range nums {
		c := Convert(n)
		if m == nil || c.Cmp(m) > 0 {
			m = c
		}
	}
	if m == nil {
		return new(apd.Decimal)
	}
	return m
}

// GetEpsilonFromDecimalPrecision mirrors getEpsilonFromDecimalPrecision
// (decimalDigits < 0 = undefined).
func GetEpsilonFromDecimalPrecision(decimalDigits int) *BN {
	if decimalDigits < 0 {
		return Convert(MedusaEpsilon)
	}
	return Div(1, Convert("1e"+strconv.Itoa(decimalDigits)))
}
