// Package httpdiff runs HTTP scenarios against a Medusa server and compares
// the responses with golden responses recorded from upstream Medusa.
//
// Record (against a running upstream server):
//
//	MEDUSA_HTTP_RECORD_URL=http://localhost:9100 go test ./integration-tests/http-diff/ -run TestRecord
package httpdiff

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Step is one request of a scenario.
type Step struct {
	Name    string            `json:"name"`
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Origin  string            `json:"origin,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    json.RawMessage   `json:"body,omitempty"`
	// Bearer sends the token returned by the named step.
	Bearer string `json:"bearer,omitempty"`
	// NoCookies sends the request without the scenario cookie jar.
	NoCookies bool `json:"nocookies,omitempty"`
}

// recordedHeaders are compared between implementations.
var recordedHeaders = []string{
	"Content-Type",
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Credentials",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Headers",
	"Vary",
}

// Response is the recorded form of a response.
type Response struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	// SetCookie lists the cookies set: name and attributes, value masked.
	SetCookie []string `json:"set_cookie,omitempty"`
	Body      any      `json:"body"`
}

func loadScenario(t *testing.T, file string) []Step {
	t.Helper()
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var steps []Step
	if err := json.Unmarshal(raw, &steps); err != nil {
		t.Fatal(err)
	}
	return steps
}

func maskCookie(header string) string {
	parts := strings.Split(header, ";")
	name, _, _ := strings.Cut(parts[0], "=")
	out := []string{name + "=<value>"}
	for _, attr := range parts[1:] {
		attr = strings.TrimSpace(attr)
		key, _, _ := strings.Cut(attr, "=")
		if strings.EqualFold(key, "expires") {
			out = append(out, "Expires=<date>")
			continue
		}
		out = append(out, attr)
	}
	return strings.Join(out, "; ")
}

// runScenario executes the steps against baseURL.
func runScenario(t *testing.T, baseURL string, steps []Step) map[string]Response {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	plain := &http.Client{CheckRedirect: client.CheckRedirect}
	out := map[string]Response{}
	tokens := map[string]string{}

	for _, step := range steps {
		var body io.Reader
		if len(step.Body) > 0 {
			body = bytes.NewReader(step.Body)
		}
		req, err := http.NewRequest(step.Method, baseURL+step.Path, body)
		if err != nil {
			t.Fatal(err)
		}
		if len(step.Body) > 0 {
			req.Header.Set("Content-Type", "application/json")
		}
		if step.Origin != "" {
			req.Header.Set("Origin", step.Origin)
		}
		for k, v := range step.Headers {
			req.Header.Set(k, v)
		}
		if step.Bearer != "" {
			req.Header.Set("Authorization", "Bearer "+tokens[step.Bearer])
		}
		c := client
		if step.NoCookies {
			c = plain
		}
		res, err := c.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", step.Name, err)
		}
		raw, _ := io.ReadAll(res.Body)
		res.Body.Close()

		rec := Response{Status: res.StatusCode, Headers: map[string]string{}}
		for _, h := range recordedHeaders {
			if v := res.Header.Values(h); len(v) > 0 {
				rec.Headers[h] = strings.Join(v, ", ")
			}
		}
		for _, c := range res.Header.Values("Set-Cookie") {
			rec.SetCookie = append(rec.SetCookie, maskCookie(c))
		}
		sort.Strings(rec.SetCookie)
		if len(raw) > 0 {
			if v, err := dal.DecodeOrderedJSON(raw); err == nil {
				rec.Body = v
				if r, ok := v.(*dal.Record); ok {
					if tok, ok := r.Get("token"); ok {
						tokens[step.Name] = fmt.Sprint(tok)
					}
				}
			} else {
				rec.Body = string(raw)
			}
		}
		out[step.Name] = rec
	}
	return out
}

func encode(v any) []byte {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
	return buf.Bytes()
}

// TestRecord records the golden responses from an upstream server.
func TestRecord(t *testing.T) {
	base := os.Getenv("MEDUSA_HTTP_RECORD_URL")
	if base == "" {
		t.Skip("MEDUSA_HTTP_RECORD_URL is not set")
	}
	files, _ := filepath.Glob("scenarios/*.json")
	for _, file := range files {
		steps := loadScenario(t, file)
		results := runScenario(t, base, steps)
		ordered := dal.NewRecord()
		for _, s := range steps {
			r := results[s.Name]
			rec := dal.NewRecord()
			rec.Set("status", float64(r.Status))
			headers := dal.NewRecord()
			for _, h := range recordedHeaders {
				if v, ok := r.Headers[h]; ok {
					headers.Set(h, v)
				}
			}
			rec.Set("headers", headers)
			if len(r.SetCookie) > 0 {
				cookies := make([]any, len(r.SetCookie))
				for i, c := range r.SetCookie {
					cookies[i] = c
				}
				rec.Set("set_cookie", cookies)
			}
			rec.Set("body", r.Body)
			ordered.Set(s.Name, rec)
		}
		raw, err := ordered.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		var pretty bytes.Buffer
		_ = json.Indent(&pretty, raw, "", "  ")
		pretty.WriteByte('\n')
		if err := os.WriteFile(filepath.Join("golden", filepath.Base(file)), pretty.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

var _ = encode

// normalize masks the values that differ between runs: JWT timestamps
// (iat/exp) and tokens (replaced by their decoded, masked payload).
func normalize(v any) any {
	switch t := v.(type) {
	case *dal.Record:
		out := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			switch {
			case k == "iat" || k == "exp":
				out.Set(k, "<timestamp>")
			case k == "token":
				out.Set(k, decodeToken(x))
			default:
				out.Set(k, normalize(x))
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = normalize(x)
		}
		return out
	}
	return v
}

func decodeToken(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return "<invalid token>"
	}
	h, _ := base64.RawURLEncoding.DecodeString(parts[0])
	p, _ := base64.RawURLEncoding.DecodeString(parts[1])
	hv, _ := dal.DecodeOrderedJSON(h)
	pv, _ := dal.DecodeOrderedJSON(p)
	out := dal.NewRecord()
	out.Set("header", hv)
	out.Set("payload", normalize(pv))
	return out
}

func asRecord(r Response) *dal.Record {
	rec := dal.NewRecord()
	rec.Set("status", float64(r.Status))
	headers := dal.NewRecord()
	for _, h := range recordedHeaders {
		if v, ok := r.Headers[h]; ok {
			headers.Set(h, v)
		}
	}
	rec.Set("headers", headers)
	if len(r.SetCookie) > 0 {
		cookies := make([]any, len(r.SetCookie))
		for i, c := range r.SetCookie {
			cookies[i] = c
		}
		rec.Set("set_cookie", cookies)
	}
	rec.Set("body", r.Body)
	return rec
}

func pretty(v any) string {
	raw, _ := dal.MarshalValue(v)
	var b bytes.Buffer
	_ = json.Indent(&b, raw, "", "  ")
	return b.String()
}

// TestMatchUpstream runs the scenarios against MEDUSA_HTTP_TARGET_URL and
// compares every response (status, headers, cookies, body with its key
// order) with the golden responses.
func TestMatchUpstream(t *testing.T) {
	base := os.Getenv("MEDUSA_HTTP_TARGET_URL")
	if base == "" {
		t.Skip("MEDUSA_HTTP_TARGET_URL is not set")
	}
	files, _ := filepath.Glob("scenarios/*.json")
	for _, file := range files {
		steps := loadScenario(t, file)
		raw, err := os.ReadFile(filepath.Join("golden", filepath.Base(file)))
		if err != nil {
			t.Fatal(err)
		}
		gv, err := dal.DecodeOrderedJSON(raw)
		if err != nil {
			t.Fatal(err)
		}
		golden := gv.(*dal.Record)
		results := runScenario(t, base, steps)
		for _, s := range steps {
			t.Run(strings.TrimSuffix(filepath.Base(file), ".json")+"/"+s.Name, func(t *testing.T) {
				want, _ := golden.Get(s.Name)
				w := pretty(normalize(want))
				g := pretty(normalize(asRecord(results[s.Name])))
				if w != g {
					t.Errorf("mismatch\n--- want\n%s\n--- got\n%s", w, g)
				}
			})
		}
	}
}
