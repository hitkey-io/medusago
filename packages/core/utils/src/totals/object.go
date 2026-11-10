package totals

import (
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Object is an ordered JavaScript-like object (dal.Record).
type Object interface {
	Keys() []string
	Get(key string) (any, bool)
	Set(key string, value any)
}

// AsObject returns v as an Object (nil when it is not one).
func AsObject(v any) Object {
	if v == nil {
		return nil
	}
	if o, ok := v.(Object); ok {
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Pointer && rv.IsNil() {
			return nil
		}
		return o
	}
	return nil
}

// Elements returns the elements of an array value ([]any, []*dal.Record,
// ...); ok is false when v is not an array.
func Elements(v any) ([]any, bool) {
	if v == nil {
		return nil, false
	}
	if a, ok := v.([]any); ok {
		return a, true
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

// Objects returns the object elements of an array value.
func Objects(v any) []Object {
	els, _ := Elements(v)
	out := make([]Object, 0, len(els))
	for _, e := range els {
		if o := AsObject(e); o != nil {
			out = append(out, o)
		}
	}
	return out
}

// Field returns o[key] (nil when absent) and whether it is defined
// (present: null counts as defined, like isDefined).
func Field(o Object, key string) (any, bool) {
	if o == nil {
		return nil, false
	}
	return o.Get(key)
}

// PickValueFromObject mirrors pickValueFromObject for object paths.
func PickValueFromObject(path string, o Object) (any, bool) {
	segments := strings.Split(path, ".")
	var cur any = o
	for i, seg := range segments {
		obj := AsObject(cur)
		if obj == nil {
			return nil, false
		}
		v, ok := obj.Get(seg)
		if !ok {
			return nil, false
		}
		if i == len(segments)-1 {
			return v, true
		}
		if v == nil {
			return nil, false
		}
		cur = v
	}
	return nil, false
}

// Truthy mirrors JavaScript truthiness for record values.
func Truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0
	case int:
		return t != 0
	}
	return true
}

// BN wraps a computed value as a BigNumber (new BigNumber(value)).
func NewBN(v any) *BigNumber {
	b, err := NewBigNumber(v)
	if err != nil {
		b, _ = NewBigNumber(0)
	}
	return b
}

// Sum mirrors MathBN.sum.
func Sum(nums ...any) *BN { return Add(append([]any{0}, nums...)...) }

func isBigNumber(v any) bool {
	_, ok := v.(*BigNumber)
	return ok
}

// TransformPropertiesToBigNumber mirrors transformPropertiesToBigNumber
// (prefix "raw_").
func TransformPropertiesToBigNumber(obj any, include, exclude []string) {
	const prefix = "raw_"
	type frame struct {
		current any
		path    string
	}
	stack := []frame{{obj, ""}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if f.current == nil || isBigNumber(f.current) {
			continue
		}
		if els, ok := Elements(f.current); ok {
			for _, e := range els {
				stack = append(stack, frame{e, f.path})
			}
			continue
		}
		o := AsObject(f.current)
		if o == nil {
			continue
		}
		for _, key := range o.Keys() {
			value, _ := o.Get(key)
			currentPath := key
			if f.path != "" {
				currentPath = f.path + "." + key
			}
			if value != nil && !contains(exclude, currentPath) {
				if strings.HasPrefix(key, prefix) {
					newKey := strings.Replace(key, prefix, "", 1)
					newPath := newKey
					if f.path != "" {
						newPath = f.path + "." + newKey
					}
					if !contains(exclude, newPath) {
						o.Set(newKey, bigNumberFromRaw(value))
						continue
					}
				} else if contains(include, currentPath) {
					o.Set(key, NewBN(value))
					continue
				}
			}
			stack = append(stack, frame{value, currentPath})
		}
	}
}

func bigNumberFromRaw(v any) *BigNumber {
	if o := AsObject(v); o != nil {
		val, _ := o.Get("value")
		prec, _ := o.Get("precision")
		p := DefaultPrecision
		if f, ok := prec.(float64); ok {
			p = int(f)
		}
		if s, ok := val.(string); ok {
			b, err := NewBigNumberWithPrecision(BigNumberRawValue{Value: s, Precision: p}, p)
			if err == nil {
				return b
			}
		}
		return NewBN(Convert(val))
	}
	return NewBN(Convert(v))
}

// CreateRawPropertiesFromBigNumber mirrors createRawPropertiesFromBigNumber
// (prefix "raw_"); newObject builds the raw value objects.
func CreateRawPropertiesFromBigNumber(obj any, newObject func() Object, exclude ...string) {
	const prefix = "raw_"
	type frame struct {
		current any
		path    string
	}
	stack := []frame{{obj, ""}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if f.current == nil || isBigNumber(f.current) || strings.Contains(f.path, "."+prefix) {
			continue
		}
		if els, ok := Elements(f.current); ok {
			for _, e := range els {
				stack = append(stack, frame{e, f.path})
			}
			continue
		}
		o := AsObject(f.current)
		if o == nil {
			continue
		}
		for _, key := range o.Keys() {
			value, _ := o.Get(key)
			currentPath := key
			if f.path != "" {
				currentPath = f.path + "." + key
			}
			if value != nil && !contains(exclude, currentPath) {
				if d, ok := value.(*BN); ok {
					value = NewBN(d)
					o.Set(key, value)
				}
				if b, ok := value.(*BigNumber); ok {
					newKey := prefix + key
					newPath := newKey
					if f.path != "" {
						newPath = f.path + "." + newKey
					}
					if !contains(exclude, newPath) {
						raw := newObject()
						raw.Set("value", TrimZeros(b.raw.Value))
						raw.Set("precision", float64(b.raw.Precision))
						o.Set(newKey, raw)
						continue
					}
				}
			}
			stack = append(stack, frame{value, currentPath})
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// JSNumberToString mirrors String(number) in JavaScript.
func JSNumberToString(f float64) string {
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
			return mant + "e+" + strings.TrimLeft(exp[1:], "0")
		}
		return mant + "e-" + strings.TrimLeft(exp[1:], "0")
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// JSONNumber mirrors a value after a JSON round trip (workflow step data):
// BigNumbers become JavaScript numbers.
func JSONNumber(v any) any {
	switch t := v.(type) {
	case *BigNumber:
		b, err := t.MarshalJSON()
		if err != nil {
			return t.Numeric()
		}
		f, _ := strconv.ParseFloat(string(b), 64)
		return f
	case *BN:
		f, _ := strconv.ParseFloat(t.Text('f'), 64)
		return f
	}
	return v
}
