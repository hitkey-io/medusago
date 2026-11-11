package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// JavaScript semantics the provider relies on (string interpolation,
// truthiness, URL serialization, fetch + Response.json, Date).

// get reads obj?.[key] (undefined: ok false).
func get(obj any, key string) (any, bool) {
	r, ok := obj.(*dal.Record)
	if !ok || r == nil {
		return nil, false
	}
	return r.Get(key)
}

// jsString mirrors String(v) (template literal interpolation).
func jsString(v any, defined bool) string {
	if !defined {
		return "undefined"
	}
	switch t := v.(type) {
	case nil:
		return "null"
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		return jsNumberString(t)
	case int:
		return strconv.Itoa(t)
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			if x != nil {
				parts[i] = jsString(x, true)
			}
		}
		return strings.Join(parts, ",")
	case *dal.Record:
		return "[object Object]"
	}
	return fmt.Sprint(v)
}

// jsNumberString mirrors Number.prototype.toString.
func jsNumberString(f float64) string {
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
		s = strings.Replace(s, "e+0", "e+", 1)
		s = strings.Replace(s, "e-0", "e-", 1)
		return s
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// truthy mirrors JavaScript truthiness.
func truthy(v any, defined bool) bool {
	if !defined {
		return false
	}
	switch t := v.(type) {
	case nil:
		return false
	case string:
		return t != ""
	case bool:
		return t
	case float64:
		return t != 0 && !math.IsNaN(t)
	}
	return true
}

// jsNumber mirrors Number(v).
func jsNumber(v any, defined bool) float64 {
	if !defined {
		return math.NaN()
	}
	switch t := v.(type) {
	case nil:
		return 0
	case bool:
		if t {
			return 1
		}
		return 0
	case float64:
		return t
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return 0
		}
		switch s {
		case "Infinity", "+Infinity":
			return math.Inf(1)
		case "-Infinity":
			return math.Inf(-1)
		}
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			n, err := strconv.ParseUint(s[2:], 16, 64)
			if err != nil {
				return math.NaN()
			}
			return float64(n)
		}
		if strings.ContainsAny(s, "xXpP_") || strings.EqualFold(s, "inf") || strings.EqualFold(s, "nan") || strings.HasSuffix(strings.ToLower(s), "infinity") {
			return math.NaN()
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			var ne *strconv.NumError
			if errors.As(err, &ne) && errors.Is(ne.Err, strconv.ErrRange) {
				return f
			}
			return math.NaN()
		}
		return f
	case []any:
		switch len(t) {
		case 0:
			return 0
		case 1:
			return jsNumber(jsString(t[0], true), true)
		}
	}
	return math.NaN()
}

// isoDateAfter mirrors new Date(Date.now() + seconds * 1000).toISOString()
// ("Invalid time value" for a NaN or out of range time).
func isoDateAfter(seconds float64) (string, error) {
	ms := float64(time.Now().UnixMilli()) + seconds*1000
	if math.IsNaN(ms) || math.IsInf(ms, 0) || math.Abs(ms) > 8.64e15 {
		return "", errors.New("Invalid time value")
	}
	t := time.UnixMilli(int64(ms)).UTC()
	year := t.Year()
	prefix := fmt.Sprintf("%04d", year)
	if year < 0 || year > 9999 {
		sign := "+"
		if year < 0 {
			sign = "-"
			year = -year
		}
		prefix = fmt.Sprintf("%s%06d", sign, year)
	}
	return prefix + t.Format("-01-02T15:04:05.000Z"), nil
}

// randomHex mirrors crypto.randomBytes(n).toString("hex").
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// formEncode mirrors the application/x-www-form-urlencoded serializer of
// URLSearchParams.
func formEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			b.WriteByte('+')
		case (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '*' || c == '-' || c == '.' || c == '_':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// searchParamsURL mirrors new URL(base) with searchParams.set calls.
func searchParamsURL(base string, params [][2]string) string {
	parts := make([]string, len(params))
	for i, p := range params {
		parts[i] = formEncode(p[0]) + "=" + formEncode(p[1])
	}
	return base + "?" + strings.Join(parts, "&")
}

// whatwgURL mirrors new URL(s).toString() for an absolute https URL built
// by string interpolation: tabs and newlines are removed, the query is
// percent-encoded with the special-query set and the fragment is dropped
// (fetch does not send it).
func whatwgURL(s string) string {
	s = strings.NewReplacer("\t", "", "\n", "", "\r", "").Replace(s)
	s = strings.Trim(s, " \x00\x01\x02\x03\x04\x05\x06\x07\x08\x0b\x0c\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f")
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	base, query, ok := strings.Cut(s, "?")
	if !ok {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c < 0x21 || c > 0x7e || c == '"' || c == '<' || c == '>' || c == '\'' {
			fmt.Fprintf(&b, "%%%02X", c)
		} else {
			b.WriteByte(c)
		}
	}
	return base + "?" + b.String()
}

// fetchResult is a fetch response: ok, status, statusText and body.
type fetchResult struct {
	ok         bool
	status     int
	statusText string
	body       []byte
}

// fetch sends a request with the default client (like the global fetch).
func fetch(req *http.Request) (fetchResult, error) {
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "*/*")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fetchResult{}, errors.New("fetch failed")
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	text := strings.TrimSpace(strings.TrimPrefix(res.Status, strconv.Itoa(res.StatusCode)))
	return fetchResult{ok: res.StatusCode >= 200 && res.StatusCode < 300, status: res.StatusCode, statusText: text, body: body}, nil
}

// json mirrors Response.json() (V8's JSON.parse errors).
func (r fetchResult) json() (any, error) {
	if err := middlewares.ValidateJSONLikeV8(string(r.body)); err != nil {
		return nil, err
	}
	return dal.DecodeOrderedJSON(r.body)
}

// readProperty mirrors obj.key on a JSON value: TypeError on null and
// undefined.
func readProperty(obj any, defined bool, key string) (any, bool, error) {
	if !defined {
		return nil, false, fmt.Errorf("Cannot read properties of undefined (reading '%s')", key)
	}
	if obj == nil {
		return nil, false, fmt.Errorf("Cannot read properties of null (reading '%s')", key)
	}
	v, ok := get(obj, key)
	return v, ok, nil
}
