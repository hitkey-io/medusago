package zod

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// CoerceNumber mirrors z.coerce.number(): the input is converted with
// `Number(input)` before the number checks (NaN is an invalid_type issue).
func CoerceNumber() *Schema {
	s := Number()
	run := s.run
	s.run = func(p *payload) {
		p.value = jsNumber(p.value)
		run(p)
	}
	return s
}

var jsDecimal = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

// jsNumber mirrors Number(v) for JSON-like values.
func jsNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case bool:
		if t {
			return 1
		}
		return 0
	case nil:
		return 0
	case string:
		return jsStringToNumber(t)
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			switch y := x.(type) {
			case nil:
			case string:
				parts[i] = y
			case *dal.Record:
				parts[i] = "[object Object]"
			default:
				parts[i] = JSString(y)
			}
		}
		return jsStringToNumber(strings.Join(parts, ","))
	}
	return math.NaN()
}

func isJSWhitespace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', '\u00a0', '\u1680', '\u2028', '\u2029', '\u202f', '\u205f', '\u3000', '\ufeff':
		return true
	}
	return r >= '\u2000' && r <= '\u200a'
}

// jsStringToNumber mirrors StringToNumber.
func jsStringToNumber(s string) float64 {
	s = strings.TrimFunc(s, isJSWhitespace)
	if s == "" {
		return 0
	}
	switch s {
	case "Infinity", "+Infinity":
		return math.Inf(1)
	case "-Infinity":
		return math.Inf(-1)
	}
	if len(s) > 2 && s[0] == '0' {
		base := 0
		switch s[1] {
		case 'x', 'X':
			base = 16
		case 'o', 'O':
			base = 8
		case 'b', 'B':
			base = 2
		}
		if base != 0 {
			n, err := strconv.ParseUint(s[2:], base, 64)
			if err != nil || strings.Contains(s[2:], "_") {
				return math.NaN()
			}
			return float64(n)
		}
	}
	if !jsDecimal.MatchString(s) {
		return math.NaN()
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return f
		}
		return math.NaN()
	}
	return f
}
