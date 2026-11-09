// Package dal mirrors @medusajs/utils/src/dal: data access for module
// models.
package dal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// Record is an entity row: an ordered set of fields, serialized to JSON in
// insertion order like a JavaScript object.
type Record struct {
	keys   []string
	values map[string]any
}

// NewRecord creates an empty record.
func NewRecord() *Record {
	return &Record{values: map[string]any{}}
}

// Set sets a field, keeping the position of an existing key.
func (r *Record) Set(key string, value any) {
	if _, ok := r.values[key]; !ok {
		r.keys = append(r.keys, key)
	}
	r.values[key] = value
}

// Get returns a field value.
func (r *Record) Get(key string) (any, bool) {
	v, ok := r.values[key]
	return v, ok
}

// Has reports whether the field is set.
func (r *Record) Has(key string) bool {
	_, ok := r.values[key]
	return ok
}

// Delete removes a field.
func (r *Record) Delete(key string) {
	if _, ok := r.values[key]; !ok {
		return
	}
	delete(r.values, key)
	for i, k := range r.keys {
		if k == key {
			r.keys = append(r.keys[:i], r.keys[i+1:]...)
			break
		}
	}
}

// Keys returns the field names in order.
func (r *Record) Keys() []string { return append([]string(nil), r.keys...) }

// Len returns the number of fields.
func (r *Record) Len() int { return len(r.keys) }

// MarshalJSON writes the fields in order. Times are written like
// Date.prototype.toJSON (UTC, millisecond precision).
func (r *Record) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, k := range r.keys {
		if i > 0 {
			b.WriteByte(',')
		}
		kb, _ := json.Marshal(k)
		b.Write(kb)
		b.WriteByte(':')
		vb, err := MarshalValue(r.values[k])
		if err != nil {
			return nil, err
		}
		b.Write(vb)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

// MarshalValue marshals a record value with JavaScript semantics.
func MarshalValue(v any) ([]byte, error) {
	switch t := v.(type) {
	case time.Time:
		return json.Marshal(FormatJSDate(t))
	case []*Record:
		var b bytes.Buffer
		b.WriteByte('[')
		for i, r := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			rb, err := r.MarshalJSON()
			if err != nil {
				return nil, err
			}
			b.Write(rb)
		}
		b.WriteByte(']')
		return b.Bytes(), nil
	default:
		return json.Marshal(v)
	}
}

// FormatJSDate formats a time like Date.prototype.toISOString.
func FormatJSDate(t time.Time) string {
	return t.UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z")
}

// OrderedJSON is a decoded JSON value whose objects keep their key order,
// used for jsonb columns.
type OrderedJSON struct{ value any }

// MarshalJSON re-encodes the value preserving key order.
func (o OrderedJSON) MarshalJSON() ([]byte, error) { return MarshalValue(o.value) }

// Value returns the decoded value: *Record for objects, []any for arrays.
func (o OrderedJSON) Value() any { return o.value }

// DecodeOrderedJSON decodes JSON keeping object key order.
func DecodeOrderedJSON(data []byte) (any, error) {
	// Numbers decode to float64, like JSON.parse.
	dec := json.NewDecoder(bytes.NewReader(data))
	v, err := decodeOrdered(dec)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func decodeOrdered(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			r := NewRecord()
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := decodeOrdered(dec)
				if err != nil {
					return nil, err
				}
				r.Set(kt.(string), v)
			}
			_, err := dec.Token()
			return r, err
		case '[':
			arr := []any{}
			for dec.More() {
				v, err := decodeOrdered(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, v)
			}
			_, err := dec.Token()
			return arr, err
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	default:
		return t, nil
	}
}
