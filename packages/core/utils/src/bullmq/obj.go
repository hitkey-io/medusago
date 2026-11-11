package bullmq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
)

// Obj is a JavaScript-like object: string keys in insertion order. Job
// options are Objs so that they are packed and serialized in the same order
// as bullmq does.
type Obj struct {
	keys []string
	vals map[string]any
}

// NewObj builds an Obj from key/value pairs.
func NewObj(kv ...any) *Obj {
	o := &Obj{vals: map[string]any{}}
	for i := 0; i+1 < len(kv); i += 2 {
		o.Set(kv[i].(string), kv[i+1])
	}
	return o
}

// ObjFromMap builds an Obj from a map (keys sorted) or returns an Obj
// unchanged; nested maps are converted too.
func ObjFromMap(v any) *Obj {
	switch m := v.(type) {
	case *Obj:
		return m
	case map[string]any:
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		o := NewObj()
		for _, k := range keys {
			o.Set(k, normalize(m[k]))
		}
		return o
	case interface {
		Keys() []string
		Get(string) (any, bool)
	}:
		o := NewObj()
		for _, k := range m.Keys() {
			val, _ := m.Get(k)
			o.Set(k, normalize(val))
		}
		return o
	}
	return nil
}

func normalize(v any) any {
	switch v.(type) {
	case map[string]any, interface {
		Keys() []string
		Get(string) (any, bool)
	}:
		return ObjFromMap(v)
	}
	return v
}

// Set sets a key, keeping the position of an existing key.
func (o *Obj) Set(k string, v any) *Obj {
	if o.vals == nil {
		o.vals = map[string]any{}
	}
	if _, ok := o.vals[k]; !ok {
		o.keys = append(o.keys, k)
	}
	o.vals[k] = v
	return o
}

// Get returns a value.
func (o *Obj) Get(k string) (any, bool) {
	if o == nil {
		return nil, false
	}
	v, ok := o.vals[k]
	return v, ok
}

// Has reports whether a key is set to a non-nil value.
func (o *Obj) Has(k string) bool {
	v, ok := o.Get(k)
	return ok && v != nil
}

// Delete removes a key.
func (o *Obj) Delete(k string) {
	if o == nil {
		return
	}
	if _, ok := o.vals[k]; !ok {
		return
	}
	delete(o.vals, k)
	for i, key := range o.keys {
		if key == k {
			o.keys = append(o.keys[:i:i], o.keys[i+1:]...)
			break
		}
	}
}

// Keys returns the keys in order.
func (o *Obj) Keys() []string {
	if o == nil {
		return nil
	}
	return append([]string(nil), o.keys...)
}

// Clone is a shallow copy ({ ...o }).
func (o *Obj) Clone() *Obj {
	c := NewObj()
	if o == nil {
		return c
	}
	for _, k := range o.keys {
		c.Set(k, o.vals[k])
	}
	return c
}

// Assign mirrors Object.assign(o, ...sources).
func (o *Obj) Assign(sources ...*Obj) *Obj {
	for _, s := range sources {
		if s == nil {
			continue
		}
		for _, k := range s.keys {
			o.Set(k, s.vals[k])
		}
	}
	return o
}

// MarshalJSON writes the keys in order; nil values are omitted like
// undefined in JSON.stringify.
func (o *Obj) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	first := true
	if o != nil {
		for _, k := range o.keys {
			v := o.vals[k]
			if v == nil {
				continue
			}
			if !first {
				b.WriteByte(',')
			}
			first = false
			kb, _ := json.Marshal(k)
			b.Write(kb)
			b.WriteByte(':')
			vb, err := jsonStringify(v)
			if err != nil {
				return nil, err
			}
			b.Write(vb)
		}
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// UnmarshalJSON decodes an object keeping the key order.
func (o *Obj) UnmarshalJSON(data []byte) error {
	v, err := parseJSON(data)
	if err != nil {
		return err
	}
	m, ok := v.(*Obj)
	if !ok {
		return fmt.Errorf("bullmq: not an object: %s", data)
	}
	*o = *m
	return nil
}

// jsonStringify mirrors JSON.stringify (no HTML escaping).
func jsonStringify(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(b.Bytes(), "\n"), nil
}

// parseJSON mirrors JSON.parse: objects as *Obj, numbers as float64.
func parseJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := parseValue(dec)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func parseValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := NewObj()
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				o.Set(kt.(string), v)
			}
			_, err := dec.Token()
			return o, err
		case '[':
			arr := []any{}
			for dec.More() {
				v, err := parseValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			_, err := dec.Token()
			return arr, err
		}
	case json.Number:
		f, err := strconv.ParseFloat(string(t), 64)
		return f, err
	}
	return tok, nil
}

// toNumber converts a JSON-like value to a float64 (NaN when not a number).
func toNumber(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case float64:
		return n
	case float32:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return math.NaN()
		}
		return f
	case bool:
		if n {
			return 1
		}
		return 0
	}
	return math.NaN()
}

// truthy mirrors JavaScript truthiness for JSON-like values.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case int, int64, int32, float64, float32, json.Number:
		f := toNumber(x)
		return f != 0 && !math.IsNaN(f)
	}
	return true
}

func numOr(o *Obj, k string, def float64) float64 {
	v, ok := o.Get(k)
	if !ok || v == nil {
		return def
	}
	f := toNumber(v)
	if math.IsNaN(f) {
		return def
	}
	return f
}

// jsString mirrors String(v) for the values used in keys.
func jsString(v any) string {
	switch x := v.(type) {
	case nil:
		return "undefined"
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	}
	f := toNumber(v)
	if math.IsNaN(f) {
		return fmt.Sprint(v)
	}
	return formatNumber(f)
}

// formatNumber mirrors Number.prototype.toString for finite numbers.
func formatNumber(f float64) string {
	if f == math.Trunc(f) && math.Abs(f) < 1e21 {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return strconv.FormatFloat(f, 'g', -1, 64)
}
