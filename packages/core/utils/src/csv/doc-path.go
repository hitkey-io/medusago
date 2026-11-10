// Package csv mirrors the CSV helpers the workflows use: json-2-csv 5.6.0
// (json2csv) with its dependencies deeks 3.2.1 (key discovery) and doc-path
// 4.1.4 (key path evaluation). Documents are JSON-like values: *dal.Record
// objects (key order kept), []any arrays, strings, float64 numbers, bools,
// nil (null) and Undefined.
package csv

import (
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

type undefinedValue struct{}

// MarshalJSON renders undefined like JSON.stringify inside an array.
func (undefinedValue) MarshalJSON() ([]byte, error) { return []byte("null"), nil }

// Undefined is the JavaScript undefined value.
var Undefined any = undefinedValue{}

func isUndefined(v any) bool { _, ok := v.(undefinedValue); return ok }

// isObject mirrors typeof v === "object" && v !== null (arrays included).
func isObject(v any) bool {
	switch v.(type) {
	case *dal.Record, []any:
		return v != (*dal.Record)(nil)
	}
	return false
}

func asArray(v any) ([]any, bool) { a, ok := v.([]any); return a, ok }

// falsy mirrors !v.
func falsy(v any) bool {
	switch t := v.(type) {
	case nil, undefinedValue:
		return true
	case bool:
		return !t
	case string:
		return t == ""
	case float64:
		return t == 0 || t != t
	case int:
		return t == 0
	case *dal.Record:
		return t == nil
	}
	return false
}

// objectKeys mirrors Object.keys.
func objectKeys(v any) []string {
	switch t := v.(type) {
	case *dal.Record:
		return t.Keys()
	case []any:
		keys := make([]string, len(t))
		for i := range t {
			keys[i] = strconv.Itoa(i)
		}
		return keys
	}
	return nil
}

// has mirrors key in obj.
func has(obj any, key string) bool {
	switch t := obj.(type) {
	case *dal.Record:
		return t.Has(key)
	case []any:
		i, err := strconv.Atoi(key)
		return err == nil && i >= 0 && i < len(t) && strconv.Itoa(i) == key
	}
	return false
}

// get mirrors obj[key] (undefined when absent).
func get(obj any, key string) any {
	switch t := obj.(type) {
	case *dal.Record:
		if v, ok := t.Get(key); ok {
			return v
		}
	case []any:
		if has(t, key) {
			i, _ := strconv.Atoi(key)
			return t[i]
		}
	}
	return Undefined
}

// parseIntIsNaN mirrors isNaN(parseInt(s)).
func parseIntIsNaN(s string) bool {
	s = strings.TrimLeft(s, jsWhitespace)
	if s != "" && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	return s == "" || s[0] < '0' || s[0] > '9'
}

func findFirstNonEscapedDotIndex(kp string) int {
	for i := 0; i < len(kp); i++ {
		if kp[i] == '.' && (i == 0 || kp[i-1] != '\\') {
			return i
		}
	}
	return -1
}

// state mirrors doc-path's state(kp).
func state(kp string) (dotIndex int, key, remaining string) {
	dotIndex = findFirstNonEscapedDotIndex(kp)
	key = kp
	if dotIndex >= 0 {
		key = kp[:dotIndex]
	}
	key = strings.ReplaceAll(key, "\\.", ".")
	remaining = kp[dotIndex+1:]
	return
}

// EvaluatePath mirrors doc-path's evaluatePath.
func EvaluatePath(obj any, kp string) any {
	if falsy(obj) {
		return nil
	}
	dotIndex, key, remaining := state(kp)
	isObj := isObject(obj)
	kpIn := isObj && has(obj, kp)
	keyIn := isObj && has(obj, key)
	kpVal, keyVal := Undefined, Undefined
	if kpIn {
		kpVal = get(obj, kp)
	}
	if keyIn {
		keyVal = get(obj, key)
	}
	if dotIndex >= 0 && isObj && !kpIn {
		_, nextKey, _ := state(remaining)
		if arr, ok := asArray(keyVal); ok && parseIntIsNaN(nextKey) {
			out := make([]any, len(arr))
			for i, doc := range arr {
				out[i] = EvaluatePath(doc, remaining)
			}
			return out
		}
		return EvaluatePath(keyVal, remaining)
	} else if arr, ok := asArray(obj); ok {
		if kp == key && dotIndex == -1 && !parseIntIsNaN(key) {
			return keyVal
		}
		out := make([]any, len(arr))
		for i, doc := range arr {
			out[i] = EvaluatePath(doc, kp)
		}
		return out
	} else if dotIndex >= 0 && kp != key && isObj && keyIn {
		return EvaluatePath(keyVal, remaining)
	} else if dotIndex == -1 && isObj && keyIn && !kpIn {
		return keyVal
	}
	return kpVal
}

// jsWhitespace is the whitespace String.prototype.trim and parseInt skip.
var jsWhitespace = " \t\n\r\v\f" + string([]rune{0xa0, 0xfeff, 0x2028, 0x2029, 0x1680, 0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007, 0x2008, 0x2009, 0x200a, 0x202f, 0x205f, 0x3000})
