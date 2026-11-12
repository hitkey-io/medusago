// Package utils mirrors @medusajs/core-flows/src/cart/utils.
package utils

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Get returns r[k] (nil when absent or r is nil).
func Get(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// Has reports whether r has the key (null counts, like isDefined on a
// present value; absent keys are undefined).
func Has(r *dal.Record, k string) bool {
	if r == nil {
		return false
	}
	v, ok := r.Get(k)
	return ok && v != nil
}

// Rec returns r[k] as a record.
func Rec(r *dal.Record, k string) *dal.Record {
	v, _ := Get(r, k).(*dal.Record)
	return v
}

// Str returns r[k] as a string.
func Str(r *dal.Record, k string) string {
	v, _ := Get(r, k).(string)
	return v
}

// Records returns the record elements of an array value.
func Records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		out := make([]*dal.Record, 0, len(t))
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok && r != nil {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// Copy returns a shallow copy of r.
func Copy(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r != nil {
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, v)
		}
	}
	return out
}

// Truthy mirrors JavaScript truthiness.
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

// IsPresent mirrors isPresent.
func IsPresent(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	case *dal.Record:
		return t != nil && t.Len() > 0
	}
	return true
}

// Coalesce mirrors a ?? b ?? ... (the first non-nil value).
func Coalesce(vs ...any) any {
	for _, v := range vs {
		if v != nil {
			return v
		}
	}
	return nil
}

// SimpleHash mirrors simpleHash (32-bit signed arithmetic, hex).
func SimpleHash(text string) string {
	hash := float64(5381)
	for _, u := range utf16(text) {
		shifted := int32(uint32(int64(math.Mod(math.Trunc(hash), 4294967296))) << 5)
		hash = float64(shifted) + hash + float64(u)
	}
	if hash < 0 {
		return "-" + strconv.FormatInt(int64(-hash), 16)
	}
	return strconv.FormatInt(int64(hash), 16)
}

func utf16(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r >= 0x10000 {
			r -= 0x10000
			out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
		} else {
			out = append(out, uint16(r))
		}
	}
	return out
}

// JSONString mirrors JSON.stringify for records.
func JSONString(v any) string {
	b, err := dal.MarshalValue(v)
	if err != nil {
		b, _ = json.Marshal(fmt.Sprint(v))
	}
	return string(b)
}
