package bullmq

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"sort"
)

// pack mirrors msgpackr's pack({ useRecords: false, encodeUndefinedAsNil:
// true }) for the JSON-like values bullmq packs (the Lua commands decode
// them with cmsgpack).
func pack(v any) []byte {
	return appendPack(nil, v)
}

func appendPack(b []byte, v any) []byte {
	switch x := v.(type) {
	case nil:
		return append(b, 0xc0)
	case bool:
		if x {
			return append(b, 0xc3)
		}
		return append(b, 0xc2)
	case string:
		return appendString(b, x)
	case int:
		return appendNumber(b, float64(x))
	case int64:
		return appendNumber(b, float64(x))
	case int32:
		return appendNumber(b, float64(x))
	case float32:
		return appendNumber(b, float64(x))
	case float64:
		return appendNumber(b, x)
	case json.Number:
		f, _ := x.Float64()
		return appendNumber(b, f)
	case []any:
		b = appendArrayHeader(b, len(x))
		for _, e := range x {
			b = appendPack(b, e)
		}
		return b
	case []string:
		b = appendArrayHeader(b, len(x))
		for _, e := range x {
			b = appendString(b, e)
		}
		return b
	case *Obj:
		if x == nil {
			return append(b, 0xc0)
		}
		b = appendMapHeader(b, len(x.keys))
		for _, k := range x.keys {
			b = appendString(b, k)
			b = appendPack(b, x.vals[k])
		}
		return b
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		b = appendMapHeader(b, len(keys))
		for _, k := range keys {
			b = appendString(b, k)
			b = appendPack(b, x[k])
		}
		return b
	}
	if o := ObjFromMap(v); o != nil {
		return appendPack(b, o)
	}
	return appendString(b, jsString(v))
}

func appendString(b []byte, s string) []byte {
	n := len(s)
	switch {
	case n < 32:
		b = append(b, 0xa0|byte(n))
	case n < 1<<8:
		b = append(b, 0xd9, byte(n))
	case n < 1<<16:
		b = append(b, 0xda)
		b = binary.BigEndian.AppendUint16(b, uint16(n))
	default:
		b = append(b, 0xdb)
		b = binary.BigEndian.AppendUint32(b, uint32(n))
	}
	return append(b, s...)
}

func appendArrayHeader(b []byte, n int) []byte {
	switch {
	case n < 16:
		return append(b, 0x90|byte(n))
	case n < 1<<16:
		b = append(b, 0xdc)
		return binary.BigEndian.AppendUint16(b, uint16(n))
	}
	b = append(b, 0xdd)
	return binary.BigEndian.AppendUint32(b, uint32(n))
}

func appendMapHeader(b []byte, n int) []byte {
	switch {
	case n < 16:
		return append(b, 0x80|byte(n))
	case n < 1<<16:
		b = append(b, 0xde)
		return binary.BigEndian.AppendUint16(b, uint16(n))
	}
	b = append(b, 0xdf)
	return binary.BigEndian.AppendUint32(b, uint32(n))
}

// appendNumber mirrors msgpackr: 32-bit integers in the smallest integer
// format, any other number as a float64.
func appendNumber(b []byte, f float64) []byte {
	if f == math.Trunc(f) && !math.IsInf(f, 0) && !(f == 0 && math.Signbit(f)) {
		switch {
		case f >= 0 && f < 0x80:
			return append(b, byte(f))
		case f >= 0 && f < 0x100:
			return append(b, 0xcc, byte(f))
		case f >= 0 && f < 0x10000:
			b = append(b, 0xcd)
			return binary.BigEndian.AppendUint16(b, uint16(f))
		case f >= 0 && f <= math.MaxUint32:
			b = append(b, 0xce)
			return binary.BigEndian.AppendUint32(b, uint32(f))
		case f < 0 && f >= -0x20:
			return append(b, byte(int8(f)))
		case f < 0 && f >= -0x80:
			return append(b, 0xd0, byte(int8(f)))
		case f < 0 && f >= -0x8000:
			b = append(b, 0xd1)
			return binary.BigEndian.AppendUint16(b, uint16(int16(f)))
		case f < 0 && f >= math.MinInt32:
			b = append(b, 0xd2)
			return binary.BigEndian.AppendUint32(b, uint32(int32(f)))
		}
	}
	b = append(b, 0xcb)
	return binary.BigEndian.AppendUint64(b, math.Float64bits(f))
}
