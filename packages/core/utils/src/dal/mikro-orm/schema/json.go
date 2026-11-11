package schema

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

// jsonNode is a JSON value with ordered object keys, written like
// JSON.stringify(value, null, 2).
type jsonNode interface{}

type jsonField struct {
	key   string
	value jsonNode
}

type jsonObject []jsonField

type jsonRaw json.RawMessage

func (o *jsonObject) set(key string, value jsonNode) {
	*o = append(*o, jsonField{key, value})
}

// stringify mirrors JSON.stringify(value, null, 2).
func stringify(v jsonNode) string {
	var b strings.Builder
	writeJSON(&b, v, "")
	return b.String()
}

func writeJSON(b *strings.Builder, v jsonNode, indent string) {
	inner := indent + "  "
	switch x := v.(type) {
	case nil:
		b.WriteString("null")
	case bool:
		b.WriteString(strconv.FormatBool(x))
	case int:
		b.WriteString(strconv.Itoa(x))
	case *int:
		if x == nil {
			b.WriteString("null")
		} else {
			b.WriteString(strconv.Itoa(*x))
		}
	case string:
		b.WriteString(quoteJSON(x))
	case *string:
		if x == nil {
			b.WriteString("null")
		} else {
			b.WriteString(quoteJSON(*x))
		}
	case []string:
		if len(x) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		for i, s := range x {
			b.WriteString(inner)
			b.WriteString(quoteJSON(s))
			if i < len(x)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(indent + "]")
	case []jsonNode:
		if len(x) == 0 {
			b.WriteString("[]")
			return
		}
		b.WriteString("[\n")
		for i, e := range x {
			b.WriteString(inner)
			writeJSON(b, e, inner)
			if i < len(x)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(indent + "]")
	case jsonObject:
		if len(x) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{\n")
		for i, f := range x {
			b.WriteString(inner)
			b.WriteString(quoteJSON(f.key))
			b.WriteString(": ")
			writeJSON(b, f.value, inner)
			if i < len(x)-1 {
				b.WriteByte(',')
			}
			b.WriteByte('\n')
		}
		b.WriteString(indent + "}")
	case jsonNumber:
		b.WriteString(string(x))
	case jsonRaw:
		writeRawJSON(b, json.RawMessage(x), indent)
	default:
		panic(fmt.Sprintf("schema: cannot stringify %T", v))
	}
}

// writeRawJSON re-indents a raw JSON value, keeping its key order.
func writeRawJSON(b *strings.Builder, raw json.RawMessage, indent string) {
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	node, err := decodeOrdered(dec)
	if err != nil {
		panic(fmt.Sprintf("schema: invalid raw JSON: %v", err))
	}
	writeJSON(b, node, indent)
}

func decodeOrdered(dec *json.Decoder) (jsonNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			obj := jsonObject{}
			for dec.More() {
				kt, err := dec.Token()
				if err != nil {
					return nil, err
				}
				v, err := decodeOrdered(dec)
				if err != nil {
					return nil, err
				}
				obj.set(kt.(string), v)
			}
			_, err := dec.Token()
			return obj, err
		case '[':
			arr := []jsonNode{}
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
	case json.Number:
		return jsonNumber(t), nil
	case string:
		return t, nil
	case bool:
		return t, nil
	case nil:
		return nil, nil
	}
	return nil, fmt.Errorf("unexpected token %v", tok)
}

// jsonNumber is a number kept as written.
type jsonNumber string

// quoteJSON mirrors the string quoting of JSON.stringify.
func quoteJSON(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			fmt.Fprintf(&b, `\u%04x`, s[i])
			i++
			continue
		}
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
		i += size
	}
	b.WriteByte('"')
	return b.String()
}
