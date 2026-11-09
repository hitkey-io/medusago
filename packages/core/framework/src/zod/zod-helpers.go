package zod

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// JSString mirrors String(value) in JavaScript.
func JSString(v any) string {
	switch t := v.(type) {
	case undefinedType:
		return "undefined"
	case nil:
		return "null"
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		return JSNumber(t)
	case int:
		return strconv.Itoa(t)
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			if x == nil || IsUndefined(x) {
				parts[i] = ""
			} else {
				parts[i] = JSString(x)
			}
		}
		return strings.Join(parts, ",")
	case *dal.Record:
		return "[object Object]"
	}
	return fmt.Sprint(v)
}

// JSNumber mirrors Number.prototype.toString.
func JSNumber(f float64) string {
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
		if exp[0] == '+' {
			exp = exp[1:]
			return mant + "e+" + strings.TrimLeft(exp, "0")
		}
		return mant + "e-" + strings.TrimLeft(exp[1:], "0")
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

func formatPath(path []any) string {
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = JSString(p)
	}
	return strings.Join(parts, ", ")
}

// getValueFromBody mirrors getValueFromBody: only plain objects are
// traversed (isObject is false for arrays), Undefined when missing.
func getValueFromBody(path []any, body any) any {
	acc, ok := body.(*dal.Record)
	if !ok || acc == nil {
		return Undefined
	}
	var cur any = acc
	for _, seg := range path {
		r, ok := cur.(*dal.Record)
		if !ok || r == nil {
			return Undefined
		}
		v, ok := r.Get(JSString(seg))
		if !ok {
			return Undefined
		}
		cur = v
	}
	return cur
}

func formatInvalidType(issues []*Issue, body any) string {
	var valid []*Issue
	for _, i := range issues {
		if i.Code == "invalid_type" {
			valid = append(valid, i)
		}
	}
	var expected []string
	for _, i := range valid {
		if !IsUndefined(getValueFromBody(i.Path, body)) {
			expected = append(expected, i.Expected)
		}
	}
	if len(expected) == 0 || len(valid) == 0 {
		return ""
	}
	first := valid[0]
	received := JSString(getValueFromBody(first.Path, body))
	return fmt.Sprintf("Expected type: '%s' for field '%s', got: '%s'", strings.Join(expected, ", "), formatPath(first.Path), received)
}

func formatRequiredField(issues []*Issue, body any) string {
	for _, i := range issues {
		if i == nil {
			continue
		}
		if (i.Code == "invalid_type" || i.Code == "invalid_value") && IsUndefined(getValueFromBody(i.Path, body)) {
			return fmt.Sprintf("Field '%s' is required", formatPath(i.Path))
		}
	}
	return ""
}

func formatUnionError(issue *Issue, body any) string {
	var issues []*Issue
	for _, group := range issue.Errors {
		for _, i := range group {
			c := *i
			c.Path = append(append([]any{}, issue.Path...), i.Path...)
			issues = append(issues, &c)
		}
	}
	if len(issues) == 0 {
		return issue.Message
	}
	if m := formatInvalidType(issues, body); m != "" {
		return m
	}
	if m := formatRequiredField(issues, body); m != "" {
		return m
	}
	return issue.Message
}

// FormatError mirrors formatError.
func FormatError(err *ZodError, body any) string {
	issues := err.Issues
	if len(issues) > 3 {
		issues = issues[:3]
	}
	var msgs []string
	for _, issue := range issues {
		var m string
		switch issue.Code {
		case "invalid_type":
			m = formatInvalidType([]*Issue{issue}, body)
			if m == "" {
				m = formatRequiredField([]*Issue{issue}, body)
			}
			if m == "" {
				m = issue.Message
			}
		case "invalid_value":
			received := getValueFromBody(issue.Path, body)
			if issue.Values != nil {
				if IsUndefined(received) {
					m = fmt.Sprintf("Field '%s' is required", formatPath(issue.Path))
				} else {
					vals := make([]string, len(issue.Values))
					for i, v := range issue.Values {
						vals[i] = JSString(v)
					}
					m = fmt.Sprintf("Expected: '%s' for field '%s', but got: '%s'", strings.Join(vals, ", "), formatPath(issue.Path), JSString(received))
				}
			} else if IsUndefined(received) {
				m = fmt.Sprintf("Field '%s' is required", formatPath(issue.Path))
			} else {
				m = issue.Message
			}
		case "invalid_union":
			m = formatUnionError(issue, body)
		case "unrecognized_keys":
			m = fmt.Sprintf("Unrecognized fields: '%s'", strings.Join(issue.Keys, ", "))
		case "too_small":
			m = fmt.Sprintf("Value for field '%s' too small, expected at least: '%s'", formatPath(issue.Path), JSString(issue.Minimum))
		case "too_big":
			m = fmt.Sprintf("Value for field '%s' too big, expected at most: '%s'", formatPath(issue.Path), JSString(issue.Maximum))
		case "not_multiple_of":
			m = fmt.Sprintf("Value for field '%s' not multiple of: '%s'", formatPath(issue.Path), JSString(issue.Divisor))
		default:
			m = issue.Message
		}
		msgs = append(msgs, m)
	}
	return strings.Join(msgs, "; ")
}

// ZodValidator mirrors zodValidator: object schemas are made strict, zod
// errors become INVALID_DATA errors with the formatted message.
func ZodValidator(schema *Schema, body any) (any, error) {
	if schema.IsObject() {
		schema = schema.Strict()
	}
	out, err := schema.Parse(body)
	if err != nil {
		var zerr *ZodError
		if errors.As(err, &zerr) {
			return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Invalid request: "+FormatError(zerr, body))
		}
		return nil, err
	}
	return out, nil
}
