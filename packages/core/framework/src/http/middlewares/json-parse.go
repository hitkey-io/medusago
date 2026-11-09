package middlewares

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

// JSONSyntaxError carries a message identical to the SyntaxError V8's
// JSON.parse throws (Node 20+), which body-parser forwards to clients.
type JSONSyntaxError struct{ Message string }

func (e *JSONSyntaxError) Error() string { return e.Message }

// ValidateJSONLikeV8 checks text with the grammar and error reporting of
// V8's JsonParser. It returns nil for valid JSON.
func ValidateJSONLikeV8(text string) error {
	p := &v8JSON{src: utf16.Encode([]rune(text))}
	return p.parse()
}

type v8JSON struct {
	src []uint16
	pos int
}

const (
	v8MaxContext    = 10
	v8MinForContext = v8MaxContext*2 + 1
	tokEOS          = "eos"
	tokNumber       = "number"
	tokString       = "string"
	tokOther        = "other"
)

type v8Fail struct{ err error }

func (p *v8JSON) parse() (err error) {
	defer func() {
		if r := recover(); r != nil {
			if f, ok := r.(v8Fail); ok {
				err = f.err
				return
			}
			panic(r)
		}
	}()
	p.skipWS()
	p.value(nil)
	p.skipWS()
	if p.pos < len(p.src) {
		p.fail("Unexpected non-whitespace character after JSON", p.pos)
	}
	return nil
}

func (p *v8JSON) lineCol(pos int) (int, int) {
	line, col := 1, 1
	for i := 0; i < pos && i < len(p.src); i++ {
		if p.src[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col
}

// fail reports an error message that carries a position.
func (p *v8JSON) fail(msg string, pos int) {
	line, col := p.lineCol(pos)
	sep := " in JSON"
	if strings.HasSuffix(msg, "after JSON") {
		sep = ""
	}
	panic(v8Fail{&JSONSyntaxError{fmt.Sprintf("%s%s at position %d (line %d column %d)", msg, sep, pos, line, col)}})
}

func (p *v8JSON) tokenAt() string {
	if p.pos >= len(p.src) {
		return tokEOS
	}
	c := p.src[p.pos]
	switch {
	case c == '"':
		return tokString
	case c == '-' || (c >= '0' && c <= '9'):
		return tokNumber
	}
	return tokOther
}

// unexpected mirrors ReportUnexpectedToken with an optional specific
// message (which takes precedence).
func (p *v8JSON) unexpected(specific string) {
	if specific != "" {
		p.fail(specific, p.pos)
	}
	switch p.tokenAt() {
	case tokEOS:
		panic(v8Fail{&JSONSyntaxError{"Unexpected end of JSON input"}})
	case tokNumber:
		p.fail("Unexpected number", p.pos)
	case tokString:
		p.fail("Unexpected string", p.pos)
	}
	src := string(utf16.Decode(p.src))
	if src == "NaN" || src == "Infinity" || src == "undefined" || src == "[object Object]" {
		panic(v8Fail{&JSONSyntaxError{fmt.Sprintf("\"%s\" is not valid JSON", src)}})
	}
	token := string(utf16.Decode(p.src[p.pos : p.pos+1]))
	n := len(p.src)
	if n <= v8MinForContext {
		panic(v8Fail{&JSONSyntaxError{fmt.Sprintf("Unexpected token '%s', \"%s\" is not valid JSON", token, src)}})
	}
	start, end := 0, n
	if p.pos >= v8MaxContext {
		start = p.pos - v8MaxContext
	}
	if p.pos+v8MaxContext < n {
		end = p.pos + v8MaxContext
	}
	ctx := string(utf16.Decode(p.src[start:end]))
	switch {
	case start == 0:
		panic(v8Fail{&JSONSyntaxError{fmt.Sprintf("Unexpected token '%s', \"%s\"... is not valid JSON", token, ctx)}})
	case end == n:
		panic(v8Fail{&JSONSyntaxError{fmt.Sprintf("Unexpected token '%s', ...\"%s\" is not valid JSON", token, ctx)}})
	default:
		panic(v8Fail{&JSONSyntaxError{fmt.Sprintf("Unexpected token '%s', ...\"%s\"... is not valid JSON", token, ctx)}})
	}
}

func (p *v8JSON) skipWS() {
	for p.pos < len(p.src) {
		switch p.src[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *v8JSON) peek() int {
	if p.pos >= len(p.src) {
		return -1
	}
	return int(p.src[p.pos])
}

func (p *v8JSON) value(specific *string) {
	switch c := p.peek(); {
	case c == '{':
		p.object()
	case c == '[':
		p.array()
	case c == '"':
		p.string()
	case c == '-' || (c >= '0' && c <= '9'):
		p.number()
	case c == 't':
		p.literal("true")
	case c == 'f':
		p.literal("false")
	case c == 'n':
		p.literal("null")
	default:
		msg := ""
		if specific != nil {
			msg = *specific
		}
		p.unexpected(msg)
	}
}

func (p *v8JSON) literal(word string) {
	for i := 0; i < len(word); i++ {
		if p.pos >= len(p.src) || p.src[p.pos] != uint16(word[i]) {
			p.unexpected("")
		}
		p.pos++
	}
}

func (p *v8JSON) object() {
	p.pos++ // {
	p.skipWS()
	if p.peek() == '}' {
		p.pos++
		return
	}
	if p.peek() != '"' {
		p.unexpected("Expected property name or '}'")
	}
	for {
		p.string()
		p.skipWS()
		if p.peek() != ':' {
			p.unexpected("Expected ':' after property name")
		}
		p.pos++
		p.skipWS()
		p.value(nil)
		p.skipWS()
		if p.peek() == ',' {
			p.pos++
			p.skipWS()
			if p.peek() != '"' {
				p.unexpected("Expected double-quoted property name")
			}
			continue
		}
		if p.peek() != '}' {
			p.unexpected("Expected ',' or '}' after property value")
		}
		p.pos++
		return
	}
}

func (p *v8JSON) array() {
	p.pos++ // [
	p.skipWS()
	if p.peek() == ']' {
		p.pos++
		return
	}
	for {
		p.value(nil)
		p.skipWS()
		if p.peek() == ',' {
			p.pos++
			p.skipWS()
			continue
		}
		if p.peek() != ']' {
			p.unexpected("Expected ',' or ']' after array element")
		}
		p.pos++
		return
	}
}

func isHex(c uint16) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func (p *v8JSON) string() {
	p.pos++ // "
	for {
		if p.pos >= len(p.src) {
			p.fail("Unterminated string", p.pos)
		}
		c := p.src[p.pos]
		switch {
		case c == '"':
			p.pos++
			return
		case c == '\\':
			p.pos++
			if p.pos >= len(p.src) {
				p.fail("Unterminated string", p.pos)
			}
			switch p.src[p.pos] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				p.pos++
			case 'u':
				for i := 1; i <= 4; i++ {
					if p.pos+i >= len(p.src) || !isHex(p.src[p.pos+i]) {
						p.fail("Bad Unicode escape", p.pos+i)
					}
				}
				p.pos += 5
			default:
				p.fail("Bad escaped character", p.pos)
			}
		case c < 0x20:
			p.fail("Bad control character in string literal", p.pos)
		default:
			p.pos++
		}
	}
}

func isDigit(c int) bool { return c >= '0' && c <= '9' }

func (p *v8JSON) number() {
	start := p.pos
	if p.peek() == '-' {
		p.pos++
		if !isDigit(p.peek()) {
			p.fail("No number after minus sign", p.pos)
		}
	}
	if p.peek() == '0' {
		p.pos++
		if isDigit(p.peek()) {
			p.unexpected("")
		}
	} else {
		for isDigit(p.peek()) {
			p.pos++
		}
	}
	if p.peek() == '.' {
		p.pos++
		if !isDigit(p.peek()) {
			p.fail("Unterminated fractional number", p.pos)
		}
		for isDigit(p.peek()) {
			p.pos++
		}
	}
	if c := p.peek(); c == 'e' || c == 'E' {
		p.pos++
		if c := p.peek(); c == '+' || c == '-' {
			p.pos++
		}
		if !isDigit(p.peek()) {
			p.fail("Exponent part is missing a number", p.pos)
		}
		for isDigit(p.peek()) {
			p.pos++
		}
	}
	_ = start
}

// strictSyntaxError mirrors body-parser's createStrictSyntaxError: the
// message of JSON.parse on the body with the offending tail replaced by
// '#', then the '#' runs replaced by the original text.
func strictSyntaxError(str string, char string) error {
	index := strings.Index(str, char)
	partial := ""
	if index != -1 {
		partial = str[:index] + strings.Repeat("#", len([]rune(str[index:])))
	}
	err := ValidateJSONLikeV8(partial)
	msg := "strict violation"
	if err != nil {
		msg = err.Error()
	}
	src := []rune(str)
	runeIndex := len([]rune(str[:max(index, 0)]))
	var b strings.Builder
	rs := []rune(msg)
	for i := 0; i < len(rs); {
		if rs[i] != '#' {
			b.WriteRune(rs[i])
			i++
			continue
		}
		j := i
		for j < len(rs) && rs[j] == '#' {
			j++
		}
		end := runeIndex + (j - i)
		if end > len(src) {
			end = len(src)
		}
		b.WriteString(string(src[runeIndex:end]))
		i = j
	}
	return &JSONSyntaxError{b.String()}
}
