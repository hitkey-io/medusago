// Package totals mirrors @medusajs/utils/src/totals.
package totals

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// DefaultPrecision mirrors BigNumber.DEFAULT_PRECISION.
const DefaultPrecision = 20

// BigNumberRawValue mirrors BigNumberRawValue: the exact value stored in the
// raw_* JSON columns.
type BigNumberRawValue struct {
	Value     string `json:"value"`
	Precision int    `json:"precision"`
}

// BigNumber mirrors the BigNumber class from big-number.ts.
type BigNumber struct {
	raw     BigNumberRawValue
	decimal apd.Decimal
}

func epsilonFromEnv(name, fallback string) float64 {
	v := os.Getenv(name)
	if v == "" {
		v = fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid %s: %q", name, v))
	}
	return f
}

// MedusaEpsilon mirrors MEDUSA_EPSILON: numeric values whose magnitude is not
// greater than it are reported as 0.
var MedusaEpsilon = epsilonFromEnv("MEDUSA_EPSILON", "0.0001")

// NewBigNumber mirrors `new BigNumber(value)` for string, number and
// BigNumberRawValue inputs.
func NewBigNumber(value any) (*BigNumber, error) {
	return NewBigNumberWithPrecision(value, DefaultPrecision)
}

// NewBigNumberWithPrecision mirrors `new BigNumber(value, { precision })`.
func NewBigNumberWithPrecision(value any, precision int) (*BigNumber, error) {
	b := &BigNumber{}
	switch v := value.(type) {
	case *BigNumber:
		*b = *v
		return b, nil
	case BigNumberRawValue:
		if _, _, err := b.decimal.SetString(v.Value); err != nil {
			return nil, invalidValue(v.Value)
		}
		if v.Precision == 0 {
			v.Precision = precision
		}
		b.raw = v
		return b, nil
	case string:
		if _, _, err := b.decimal.SetString(v); err != nil {
			return nil, invalidValue(v)
		}
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, invalidValue(v)
		}
		// bignumber.js builds from the shortest round-trip representation.
		b.decimal.SetString(strconv.FormatFloat(v, 'g', -1, 64))
	case int:
		b.decimal.SetInt64(int64(v))
	case int64:
		b.decimal.SetInt64(v)
	default:
		return nil, invalidValue(value)
	}
	b.raw = BigNumberRawValue{Value: toPrecision(&b.decimal, precision), Precision: precision}
	return b, nil
}

func invalidValue(v any) error {
	return fmt.Errorf("Invalid BigNumber value: %v. Should be one of: string, number, BigNumber (bignumber.js), BigNumberRawValue", v)
}

// Raw returns the raw value.
func (b *BigNumber) Raw() BigNumberRawValue { return b.raw }

// Numeric mirrors the `numeric` getter: the raw value as a JavaScript number,
// with values within MEDUSA_EPSILON of zero reported as 0.
func (b *BigNumber) Numeric() float64 {
	f, err := strconv.ParseFloat(b.raw.Value, 64)
	if err != nil {
		f, _ = b.decimal.Float64()
	}
	if math.Abs(f) <= MedusaEpsilon {
		return 0
	}
	return f
}

// MarshalJSON mirrors toJSON: a BigNumber serializes as a number.
func (b *BigNumber) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.Numeric())
}

// String mirrors toString.
func (b *BigNumber) String() string { return b.raw.Value }

// TrimmedRaw returns the raw value as stored by the MikroORM BigNumber
// property: the value with trailing fractional zeros removed.
func (b *BigNumber) TrimmedRaw() BigNumberRawValue {
	return BigNumberRawValue{Value: TrimZeros(b.raw.Value), Precision: b.raw.Precision}
}

// TrimZeros mirrors trimZeros from trim-zeros.ts.
func TrimZeros(value string) string {
	whole, fraction, ok := strings.Cut(value, ".")
	if !ok || fraction == "" {
		return whole
	}
	decimal, exp, hasExp := strings.Cut(fraction, "e")
	decimal = strings.TrimRight(decimal, "0")
	expStr := ""
	if hasExp {
		expStr = "e" + exp
	}
	if decimal == "" {
		return whole + expStr
	}
	return whole + "." + decimal + expStr
}

// toPrecision mirrors bignumber.js toPrecision(sd) with ROUND_HALF_UP.
func toPrecision(d *apd.Decimal, sd int) string {
	if d.IsZero() {
		return "0." + strings.Repeat("0", sd-1)
	}

	ctx := apd.Context{Precision: uint32(sd), Rounding: apd.RoundHalfUp, MaxExponent: apd.MaxExponent, MinExponent: apd.MinExponent}
	var r apd.Decimal
	if _, err := ctx.Round(&r, d); err != nil {
		panic(err)
	}

	digits := r.Coeff.String()
	// Adjusted exponent: position of the most significant digit.
	exp := int(r.Exponent) + len(digits) - 1
	if len(digits) < sd {
		digits += strings.Repeat("0", sd-len(digits))
	}
	digits = digits[:sd]

	sign := ""
	if r.Negative {
		sign = "-"
	}

	if exp < -6 || exp >= sd {
		mantissa := digits[:1]
		if sd > 1 {
			mantissa += "." + digits[1:]
		}
		expSign := "+"
		if exp < 0 {
			expSign = "-"
		}
		return fmt.Sprintf("%s%se%s%d", sign, mantissa, expSign, abs(exp))
	}
	if exp < 0 {
		return sign + "0." + strings.Repeat("0", -exp-1) + digits
	}
	if exp+1 == len(digits) {
		return sign + digits
	}
	return sign + digits[:exp+1] + "." + digits[exp+1:]
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// JSNumberToPrecision mirrors Number.prototype.toPrecision for a float64
// (exact, correctly rounded digits; exponential when the exponent is < -6 or
// >= sd), without trimming zeros.
func JSNumberToPrecision(f float64, sd int) string {
	var d apd.Decimal
	if _, _, err := d.SetString(strconv.FormatFloat(f, 'e', sd-1, 64)); err != nil {
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return toPrecision(&d, sd)
}
