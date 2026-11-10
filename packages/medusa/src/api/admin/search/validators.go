package search

import (
	"math"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// coerceNumber mirrors z.coerce.number(): Number(input).
func coerceNumber(v any) any {
	switch t := v.(type) {
	case float64:
		return t
	case bool:
		if t {
			return float64(1)
		}
		return float64(0)
	case nil:
		return float64(0)
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return float64(0)
		}
		switch s {
		case "Infinity", "+Infinity":
			return math.Inf(1)
		case "-Infinity":
			return math.Inf(-1)
		}
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			if n, err := strconv.ParseUint(s[2:], 16, 64); err == nil {
				return float64(n)
			}
			return math.NaN()
		}
		if strings.ContainsAny(s, "_xXpP") || strings.EqualFold(s, "nan") || strings.EqualFold(s, "inf") || strings.EqualFold(s, "infinity") {
			return math.NaN()
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return math.NaN()
		}
		return f
	case []any:
		if len(t) == 0 {
			return float64(0)
		}
		if len(t) == 1 {
			return coerceNumber(t[0])
		}
		return math.NaN()
	}
	return math.NaN()
}

// entityParam mirrors the `entity` param: one string ("a,b" splits) or an
// array, trimmed, empty entries dropped.
var entityParam = zod.Union(zod.String(), zod.Array(zod.String())).Transform(func(v any) (any, error) {
	var items []string
	switch t := v.(type) {
	case string:
		items = strings.Split(t, ",")
	case []any:
		for _, e := range t {
			s, _ := e.(string)
			items = append(items, s)
		}
	}
	out := []any{}
	for _, e := range items {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out, nil
})

// AdminGetSearchParams mirrors AdminGetSearchParams.
var AdminGetSearchParams = zod.Object(
	zod.F("q", zod.String().Optional()),
	zod.F("entity", entityParam.Optional()),
	zod.F("limit", zod.Preprocess(coerceNumber, zod.Number().Int().Gt(0).Max(100)).Optional()),
	zod.F("offset", zod.Preprocess(coerceNumber, zod.Number().Int().Nonnegative()).Optional()),
).Strict()
