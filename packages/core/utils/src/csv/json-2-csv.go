package csv

import (
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/totals"
)

// Json2CsvOptions mirrors the json2csv options the workflows use (defaults
// as in json-2-csv's defaultJson2CsvOptions; unwindArrays, keys objects,
// checkSchemaDifferences and locale formatting are not supported).
type Json2CsvOptions struct {
	ArrayIndexesAsKeys  bool
	ExpandNestedObjects bool
	ExpandArrayObjects  bool
	PrependHeader       bool
	PreventCsvInjection bool
	SortHeader          bool
	// EmptyFieldValue nil = undefined.
	EmptyFieldValue *string
	// Keys are the header fields to use (nil = discovered).
	Keys        []string
	ExcludeKeys []string
}

// DefaultJson2CsvOptions mirrors defaultJson2CsvOptions.
func DefaultJson2CsvOptions() Json2CsvOptions {
	return Json2CsvOptions{ExpandNestedObjects: true, PrependHeader: true}
}

const (
	fieldDelimiter = ","
	wrapDelimiter  = `"`
	eolDelimiter   = "\n"
)

var (
	crlfSearchRegex   = regexp.MustCompile(`\r?\n|\r`)
	csvInjectionRegex = regexp.MustCompile(`^[ \t\r]*[=+\-@\t\r]+`)
)

func itoa(i int) string { return strconv.Itoa(i) }

// Json2Csv mirrors json2csv(data, options).
func Json2Csv(data []any, o Json2CsvOptions) string {
	deeks := deeksOptions{
		arrayIndexesAsKeys:             o.ArrayIndexesAsKeys,
		expandNestedObjects:            o.ExpandNestedObjects,
		expandArrayObjects:             o.ExpandArrayObjects,
		ignoreEmptyArraysWhenExpanding: o.ExpandArrayObjects,
		escapeNestedDots:               true,
	}
	headerFields := retrieveHeaderFields(data, o, deeks)
	rows := make([]string, len(data))
	for i, record := range data {
		values := make([]string, len(headerFields))
		for j, field := range headerFields {
			v := retrieveRecordFieldValue(record, field, o)
			if o.PreventCsvInjection {
				v = preventCsvInjection(v)
			}
			values[j] = wrapFieldValueIfNecessary(recordFieldValueToString(v))
		}
		rows[i] = strings.Join(values, fieldDelimiter)
	}
	header := ""
	if o.PrependHeader {
		wrapped := make([]string, len(headerFields))
		for i, f := range headerFields {
			wrapped[i] = wrapFieldValueIfNecessary(f)
		}
		header = strings.Join(wrapped, fieldDelimiter) + eolDelimiter
	}
	return header + strings.Join(rows, eolDelimiter)
}

func retrieveHeaderFields(data []any, o Json2CsvOptions, deeks deeksOptions) []string {
	var fields []string
	if o.Keys != nil {
		fields = append([]string{}, o.Keys...)
	} else {
		schemas := deepKeysFromList(data, deeks, nil)
		fields = []string{}
		if len(schemas) > 0 {
			fields = unique(flatten(schemas))
		}
	}
	fields = filterExcludedKeys(fields, o.ExcludeKeys)
	if o.SortHeader {
		sort.Strings(fields)
	}
	return fields
}

func filterExcludedKeys(keyPaths, excluded []string) []string {
	if len(excluded) == 0 {
		return keyPaths
	}
	out := []string{}
outer:
	for _, kp := range keyPaths {
		for _, ex := range excluded {
			if ex == kp || strings.HasPrefix(kp, ex) {
				continue outer
			}
		}
		out = append(out, kp)
	}
	return out
}

func isEmptyField(v any) bool {
	if v == nil || isUndefined(v) {
		return true
	}
	s, ok := v.(string)
	return ok && s == ""
}

func retrieveRecordFieldValue(record any, field string, o Json2CsvOptions) any {
	v := EvaluatePath(record, field)
	if o.EmptyFieldValue != nil && isEmptyField(v) {
		return *o.EmptyFieldValue
	}
	if arr, ok := asArray(v); ok && o.ExpandArrayObjects {
		filtered := []any{}
		for _, x := range arr {
			if !isEmptyField(x) {
				filtered = append(filtered, x)
			}
		}
		switch {
		case len(arr) == 0 || len(filtered) == 0:
			if o.EmptyFieldValue != nil && *o.EmptyFieldValue != "" {
				return *o.EmptyFieldValue
			}
			return ""
		case len(filtered) == 1:
			return filtered[0]
		}
		return arr
	}
	return v
}

// jsNumberIsNaN mirrors isNaN(Number(s)).
func jsNumberIsNaN(s string) bool {
	t := strings.Trim(s, jsWhitespace)
	if t == "" {
		return false
	}
	switch t {
	case "Infinity", "+Infinity", "-Infinity":
		return false
	}
	lower := strings.ToLower(t)
	for _, p := range []string{"0x", "0o", "0b"} {
		if strings.HasPrefix(lower, p) {
			base := map[string]int{"0x": 16, "0o": 8, "0b": 2}[p]
			_, err := strconv.ParseUint(t[2:], base, 64)
			return err != nil && !strings.Contains(err.Error(), "out of range")
		}
	}
	if strings.ContainsAny(lower, "_xpni") {
		return true
	}
	_, err := strconv.ParseFloat(t, 64)
	return err != nil && !strings.Contains(err.Error(), "out of range")
}

func preventCsvInjection(v any) any {
	switch t := v.(type) {
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = preventCsvInjection(x)
		}
		return out
	case string:
		if jsNumberIsNaN(t) {
			return csvInjectionRegex.ReplaceAllString(t, "")
		}
	}
	return v
}

// recordFieldValueToString mirrors recordFieldValueToString.
func recordFieldValueToString(v any) string {
	switch t := v.(type) {
	case nil:
		return "null"
	case undefinedValue:
		return "undefined"
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return totals.JSNumberToString(t)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case time.Time:
		return JSDateString(t)
	}
	b, err := dal.MarshalValue(v)
	if err != nil {
		return ""
	}
	return string(b)
}

func wrapFieldValueIfNecessary(v string) string {
	if strings.Contains(v, wrapDelimiter) {
		v = strings.ReplaceAll(v, wrapDelimiter, wrapDelimiter+wrapDelimiter)
	}
	if strings.Contains(v, fieldDelimiter) || strings.Contains(v, wrapDelimiter) || crlfSearchRegex.MatchString(v) {
		v = wrapDelimiter + v + wrapDelimiter
	}
	return v
}

var _ = math.NaN

// JSDateString mirrors Date.prototype.toString in the process time zone:
// "Tue Nov 10 2026 00:54:05 GMT+0500 (<zone name>)". The zone name is the
// ICU long name in the system locale upstream; here it is Go's zone name.
func JSDateString(t time.Time) string {
	l := t.In(time.Local)
	name, _ := l.Zone()
	return l.Format("Mon Jan 02 2006 15:04:05 GMT-0700") + " (" + name + ")"
}
