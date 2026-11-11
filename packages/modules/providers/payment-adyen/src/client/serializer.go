package client

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

var primitives = map[string]bool{
	"string": true, "boolean": true, "double": true, "integer": true, "long": true, "float": true, "number": true, "any": true,
}

const (
	nullableSuffix = " | null"
	optionalSuffix = " | undefined"
	arrayPrefix    = "Array<"
	setPrefix      = "Set<"
	mapPrefix      = "{ [key: string]: "
	mapSuffix      = "; }"
)

func copyShallow(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

// findCorrectType mirrors ObjectSerializer.findCorrectType.
func findCorrectType(data any, expected string) string {
	m := TypeMap[expected]
	if data == nil || m == nil || m.Discriminator == "" {
		return expected
	}
	r, _ := data.(*dal.Record)
	d, _ := get(r, m.Discriminator).(string)
	if d == "" {
		return expected
	}
	if t, ok := m.Mapping[d]; ok {
		return t
	}
	if TypeMap[d] != nil {
		return d
	}
	return expected
}

// Serialize mirrors ObjectSerializer.serialize: model objects keep the keys
// of their attributeTypeMap, in its order.
func Serialize(data any, typ string, format string) any {
	if data == nil {
		return nil
	}
	switch {
	case primitives[strings.ToLower(typ)]:
		return data
	case strings.HasSuffix(typ, nullableSuffix):
		return Serialize(data, strings.TrimSuffix(typ, nullableSuffix), format)
	case strings.HasSuffix(typ, optionalSuffix):
		return Serialize(data, strings.TrimSuffix(typ, optionalSuffix), format)
	case strings.HasPrefix(typ, arrayPrefix), strings.HasPrefix(typ, setPrefix):
		sub := typ[strings.Index(typ, "<")+1 : len(typ)-1]
		l, _ := data.([]any)
		out := make([]any, 0, len(l))
		for _, x := range l {
			out = append(out, Serialize(x, sub, format))
		}
		return out
	case strings.HasPrefix(typ, mapPrefix):
		sub := strings.TrimSuffix(strings.TrimPrefix(typ, mapPrefix), mapSuffix)
		r, ok := data.(*dal.Record)
		if !ok {
			return data
		}
		out := dal.NewRecord()
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, Serialize(v, sub, format))
		}
		return out
	case typ == "Date":
		return data
	}
	if strings.Contains(typ, ".") {
		return data // enum
	}
	if TypeMap[typ] == nil {
		return data
	}
	typ = findCorrectType(data, typ)
	m := TypeMap[typ]
	r, ok := data.(*dal.Record)
	if !ok {
		return data
	}
	if m.NoAttributes {
		return copyShallow(r)
	}
	out := dal.NewRecord()
	for _, a := range m.Attrs {
		v, present := r.Get(a.Name)
		if !present {
			// undefined: dropped by JSON.stringify
			continue
		}
		out.Set(a.BaseName, Serialize(v, a.Type, a.Format))
	}
	return out
}

// Deserialize mirrors ObjectSerializer.deserialize: model objects keep the
// known attributes of the data, in attributeTypeMap order.
func Deserialize(data any, typ string, format string) any {
	typ = findCorrectType(data, typ)
	if data == nil {
		return data
	}
	switch {
	case primitives[strings.ToLower(typ)]:
		return data
	case strings.HasSuffix(typ, nullableSuffix):
		return Deserialize(data, strings.TrimSuffix(typ, nullableSuffix), format)
	case strings.HasSuffix(typ, optionalSuffix):
		return Deserialize(data, strings.TrimSuffix(typ, optionalSuffix), format)
	case strings.HasPrefix(typ, arrayPrefix), strings.HasPrefix(typ, setPrefix):
		sub := typ[strings.Index(typ, "<")+1 : len(typ)-1]
		l, _ := data.([]any)
		out := make([]any, 0, len(l))
		for _, x := range l {
			out = append(out, Deserialize(x, sub, format))
		}
		return out
	case strings.HasPrefix(typ, mapPrefix):
		sub := strings.TrimSuffix(strings.TrimPrefix(typ, mapPrefix), mapSuffix)
		r, ok := data.(*dal.Record)
		if !ok {
			return data
		}
		out := dal.NewRecord()
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			out.Set(k, Deserialize(v, sub, format))
		}
		return out
	case typ == "Date":
		// new Date(data), serialized later with toISOString
		if s, ok := data.(string); ok {
			if t, ok := common.ParseJSDate(s); ok {
				return dal.FormatJSDate(t)
			}
		}
		return data
	}
	if strings.Contains(typ, ".") {
		return data // enum
	}
	m := TypeMap[typ]
	if m == nil {
		return data
	}
	r, _ := data.(*dal.Record)
	if m.NoAttributes {
		return copyShallow(r)
	}
	out := dal.NewRecord()
	if r == nil {
		return out
	}
	for _, a := range m.Attrs {
		v, present := r.Get(a.BaseName)
		if !present {
			continue
		}
		out.Set(a.Name, Deserialize(v, a.Type, a.Format))
	}
	return out
}
