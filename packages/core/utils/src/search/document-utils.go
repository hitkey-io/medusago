// Package search mirrors @medusajs/utils/src/search: defineSearchIndex,
// the graph seed/consume helpers and the document helpers shared by the
// search providers.
package search

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/csv"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// get reads a property of a JavaScript-like object value (undefined: false).
func get(v any, key string) (any, bool) {
	if r, ok := v.(*dal.Record); ok && r != nil {
		return r.Get(key)
	}
	return nil, false
}

// asArray returns the elements of an array value.
func asArray(v any) ([]any, bool) {
	switch t := v.(type) {
	case []any:
		return t, true
	case []*dal.Record:
		out := make([]any, len(t))
		for i, r := range t {
			out[i] = r
		}
		return out, true
	case []string:
		out := make([]any, len(t))
		for i, s := range t {
			out[i] = s
		}
		return out, true
	}
	return nil, false
}

func flatInfinity(values []any, out []any) []any {
	for _, v := range values {
		if arr, ok := asArray(v); ok {
			out = flatInfinity(arr, out)
			continue
		}
		out = append(out, v)
	}
	return out
}

// ReadDocumentPath mirrors readDocumentPath: walks a dotted path,
// flattening through the arrays met on the way.
func ReadDocumentPath(source any, segments []string) []any {
	current := []any{source}
	for _, segment := range segments {
		var next []any
		for _, value := range current {
			if value == nil {
				continue
			}
			if arr, ok := asArray(value); ok {
				for _, entry := range arr {
					if resolved, ok := get(entry, segment); ok {
						next = append(next, resolved)
					}
				}
				continue
			}
			if resolved, ok := get(value, segment); ok {
				next = append(next, resolved)
			}
		}
		current = next
	}
	flat := flatInfinity(current, nil)
	out := []any{}
	for _, v := range flat {
		if v != nil {
			out = append(out, v)
		}
	}
	return out
}

// SetDocumentPath mirrors setDocumentPath: writes a value at a dotted path,
// creating the intermediate objects.
func SetDocumentPath(target *dal.Record, path string, value any) {
	segments := strings.Split(path, ".")
	cursor := target
	for _, segment := range segments[:len(segments)-1] {
		v, ok := cursor.Get(segment)
		next, isRecord := v.(*dal.Record)
		if !ok || v == nil || !isRecord {
			if !ok || v == nil {
				next = dal.NewRecord()
				cursor.Set(segment, next)
			} else {
				next = dal.NewRecord()
			}
		}
		cursor = next
	}
	cursor.Set(segments[len(segments)-1], value)
}

// JSNumber mirrors Number(value).
func JSNumber(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case bool:
		if t {
			return 1
		}
		return 0
	case nil:
		return 0
	case time.Time:
		return float64(t.UnixMilli())
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
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
			return 0
		}
		if len(t) == 1 {
			return JSNumber(JSString(t[0]))
		}
		return math.NaN()
	}
	return math.NaN()
}

// JSNumberString mirrors String(number).
func JSNumberString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	}
	abs := math.Abs(f)
	if abs >= 1e21 || abs < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		mant, exp, _ := strings.Cut(s, "e")
		if strings.HasPrefix(exp, "+") {
			exp = "+" + strings.TrimLeft(exp[1:], "0")
		} else if strings.HasPrefix(exp, "-") {
			exp = "-" + strings.TrimLeft(exp[1:], "0")
		}
		return mant + "e" + exp
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// JSString mirrors String(value).
func JSString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return JSNumberString(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case bool:
		if t {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	case time.Time:
		return csv.JSDateString(t)
	case *dal.Record:
		return "[object Object]"
	}
	if arr, ok := asArray(v); ok {
		parts := make([]string, len(arr))
		for i, e := range arr {
			if e == nil {
				continue
			}
			parts[i] = JSString(e)
		}
		return strings.Join(parts, ",")
	}
	return ""
}

// SearchValueToNumber mirrors searchValueToNumber (nil: NaN).
func SearchValueToNumber(value any, isDate bool) *float64 {
	var num float64
	if isDate {
		if t, ok := value.(time.Time); ok {
			num = float64(t.UnixMilli())
		} else if t, ok := common.ParseJSDate(JSString(value)); ok {
			num = float64(t.UnixMilli())
		} else {
			num = math.NaN()
		}
	} else {
		num = JSNumber(value)
	}
	if math.IsNaN(num) {
		return nil
	}
	return &num
}

// Truthy mirrors Boolean(value).
func Truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0 && !math.IsNaN(t)
	case int:
		return t != 0
	}
	return true
}

// SearchValueToBoolean mirrors searchValueToBoolean.
func SearchValueToBoolean(value any) bool {
	if b, ok := value.(bool); ok {
		return b
	}
	if s, ok := value.(string); ok && (s == "true" || s == "false") {
		return s == "true"
	}
	return Truthy(value)
}

// ProjectDocument mirrors projectDocument: narrows a document to the
// dotted paths, keeping the shape of intermediate objects and arrays.
func ProjectDocument(source *dal.Record, paths []string) *dal.Record {
	picked := dal.NewRecord()
	for _, path := range paths {
		segments := strings.Split(path, ".")
		head, rest := segments[0], segments[1:]
		var value any
		ok := false
		if source != nil {
			value, ok = source.Get(head)
		}
		if !ok {
			continue
		}
		if len(rest) == 0 {
			picked.Set(head, value)
			continue
		}
		restPath := strings.Join(rest, ".")
		if arr, isArr := asArray(value); isArr {
			existingV, _ := picked.Get(head)
			existing, _ := existingV.([]any)
			out := make([]any, len(arr))
			for i, entry := range arr {
				var target *dal.Record
				if i < len(existing) {
					target, _ = existing[i].(*dal.Record)
				}
				if target == nil {
					target = dal.NewRecord()
				}
				entryRec, _ := entry.(*dal.Record)
				projected := ProjectDocument(entryRec, []string{restPath})
				for _, k := range projected.Keys() {
					v, _ := projected.Get(k)
					target.Set(k, v)
				}
				out[i] = target
			}
			picked.Set(head, out)
		} else if rec, isRec := value.(*dal.Record); isRec && rec != nil {
			existingV, _ := picked.Get(head)
			target, _ := existingV.(*dal.Record)
			if target == nil {
				target = dal.NewRecord()
			}
			projected := ProjectDocument(rec, []string{restPath})
			for _, k := range projected.Keys() {
				v, _ := projected.Get(k)
				target.Set(k, v)
			}
			picked.Set(head, target)
		}
	}
	return picked
}

// ExtractPrimaryKeyFilter mirrors extractPrimaryKeyFilter: the ids of a
// filter that is nothing but primary key membership (nil otherwise; an
// empty slice for an empty list).
func ExtractPrimaryKeyFilter(filters *dal.Record, primaryKey string) ([]string, bool) {
	keys := filters.Keys()
	if len(keys) != 1 || keys[0] != primaryKey {
		return nil, false
	}
	predicate, _ := filters.Get(primaryKey)
	asIDs := func(value any) ([]string, bool) {
		values, ok := asArray(value)
		if !ok {
			values = []any{value}
		}
		out := make([]string, 0, len(values))
		for _, v := range values {
			s, isString := v.(string)
			if !isString {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	rec, isRec := predicate.(*dal.Record)
	if !isRec {
		return asIDs(predicate)
	}
	operators := rec.Keys()
	if len(operators) != 1 {
		return nil, false
	}
	if operators[0] == "$eq" || operators[0] == "$in" {
		v, _ := rec.Get(operators[0])
		return asIDs(v)
	}
	return nil, false
}
