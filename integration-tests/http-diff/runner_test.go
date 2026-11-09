// Package httpdiff runs HTTP scenarios against a Medusa server and compares
// the responses with golden responses recorded from upstream Medusa.
//
// Record (against a running upstream server):
//
//	MEDUSA_HTTP_RECORD_URL=http://localhost:9100 go test ./integration-tests/http-diff/ -run TestRecord
package httpdiff

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/loaders"
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
	// Unordered lists body keys whose arrays upstream returns in a
	// nondeterministic order: they are compared sorted.
	Unordered []string `json:"unordered,omitempty"`
	// Files sends a multipart/form-data body with these files.
	Files []StepFile `json:"files,omitempty"`
}

// StepFile is one file of a multipart step.
type StepFile struct {
	Field   string `json:"field"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
}

// multipartBody builds the multipart/form-data body of the step files.
func multipartBody(files []StepFile) (io.Reader, string) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for _, f := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, f.Field, f.Name))
		h.Set("Content-Type", f.Type)
		part, _ := w.CreatePart(h)
		_, _ = part.Write([]byte(f.Content))
	}
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

// fileTimestampRe matches the Date.now() prefix of local file keys.
var fileTimestampRe = regexp.MustCompile(`\b1\d{12}-`)

// sortUnordered sorts the listed body arrays by their JSON encoding.
func sortUnordered(rec any, keys []string) any {
	r, ok := rec.(*dal.Record)
	if !ok || len(keys) == 0 {
		return rec
	}
	body, _ := r.Get("body")
	b, ok := body.(*dal.Record)
	if !ok {
		return rec
	}
	for _, k := range keys {
		v, _ := b.Get(k)
		arr, ok := v.([]any)
		if !ok {
			continue
		}
		sorted := append([]any{}, arr...)
		sort.SliceStable(sorted, func(i, j int) bool {
			a, _ := dal.MarshalValue(sorted[i])
			c, _ := dal.MarshalValue(sorted[j])
			return string(a) < string(c)
		})
		b.Set(k, sorted)
	}
	return rec
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

	bodies := map[string]any{}
	for _, step := range steps {
		step.Path = resolveRefs(step.Path, bodies, true)
		var body io.Reader
		if len(step.Body) > 0 {
			body = strings.NewReader(resolveRefs(string(step.Body), bodies, false))
		}
		contentType := ""
		if len(step.Body) > 0 {
			contentType = "application/json"
		}
		if len(step.Files) > 0 {
			body, contentType = multipartBody(step.Files)
		}
		req, err := http.NewRequest(step.Method, baseURL+step.Path, body)
		if err != nil {
			t.Fatal(err)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
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
		bodies[step.Name] = rec.Body
	}
	return out
}

var refRe = regexp.MustCompile(`\{\{([A-Za-z0-9_]+)\.([^}]+)\}\}`)
var quotedRefRe = regexp.MustCompile(`"\{\{([A-Za-z0-9_]+)\.([^}]+)\}\}"`)

// resolveRefs replaces {{step.path.to.value}} with a value of a previous
// response body (path segments: object keys or array indexes). In bodies
// the placeholder is a JSON string ("{{...}}") standing for a whole value,
// replaced by the JSON-encoded value; in paths it is inserted as text,
// URL-escaped.
func resolveRefs(s string, bodies map[string]any, inPath bool) string {
	re := refRe
	if !inPath {
		re = quotedRefRe
	}
	return re.ReplaceAllStringFunc(s, func(m string) string {
		sm := re.FindStringSubmatch(m)
		var cur any = bodies[sm[1]]
		for _, seg := range strings.Split(sm[2], ".") {
			switch t := cur.(type) {
			case *dal.Record:
				cur, _ = t.Get(seg)
			case []any:
				i := 0
				fmt.Sscan(seg, &i)
				if i < len(t) {
					cur = t[i]
				} else {
					cur = nil
				}
			default:
				cur = nil
			}
		}
		if inPath {
			return url.QueryEscape(fmt.Sprint(cur))
		}
		raw, _ := dal.MarshalValue(cur)
		return string(raw)
	})
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
	for _, file := range scenarioFiles() {
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

var embeddedIDRe = regexp.MustCompile(`[a-z]+_[0-9A-HJKMNP-TV-Z]{26}`)

// placeholders maps generated values to stable names within a scenario.
type placeholders struct {
	names map[string]string
	count map[string]int
}

func newPlaceholders() *placeholders {
	return &placeholders{names: map[string]string{}, count: map[string]int{}}
}

func (p *placeholders) of(s string) string {
	s = fileTimestampRe.ReplaceAllString(s, "<ms>-")
	gen, kind := isGenerated(s)
	if !gen {
		if kind == "" && len(s) > 30 {
			// generated ids inside messages
			return embeddedIDRe.ReplaceAllStringFunc(s, func(id string) string {
				if g, k := isGenerated(id); g && k == "id" {
					return p.of(id)
				}
				return id
			})
		}
		return s
	}
	if kind == "date" {
		// which rows share a millisecond depends on timing (upstream too)
		return "<date>"
	}
	if n, ok := p.names[s]; ok {
		return n
	}
	p.count[kind]++
	n := fmt.Sprintf("<%s %d>", kind, p.count[kind])
	p.names[s] = n
	return n
}

// normalize masks the values that differ between runs: JWT timestamps
// (iat/exp), tokens (replaced by their decoded, masked payload) and ids and
// timestamps generated during the scenario.
func (p *placeholders) normalize(v any) any {
	switch t := v.(type) {
	case string:
		return p.of(t)
	case *dal.Record:
		out := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			switch {
			case k == "iat" || k == "exp" || k == "jti":
				out.Set(k, "<timestamp>")
			case k == "token":
				out.Set(k, p.decodeToken(x))
			default:
				out.Set(k, p.normalize(x))
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = p.normalize(x)
		}
		return out
	}
	return v
}

func normalize(v any) any {
	switch t := v.(type) {
	case *dal.Record:
		out := dal.NewRecord()
		for _, k := range t.Keys() {
			x, _ := t.Get(k)
			switch {
			case k == "iat" || k == "exp" || k == "jti":
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

func (p *placeholders) decodeToken(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return "<invalid token>"
	}
	h, _ := base64.RawURLEncoding.DecodeString(parts[0])
	pl, _ := base64.RawURLEncoding.DecodeString(parts[1])
	hv, _ := dal.DecodeOrderedJSON(h)
	pv, _ := dal.DecodeOrderedJSON(pl)
	out := dal.NewRecord()
	out.Set("header", hv)
	out.Set("payload", p.normalize(pv))
	return out
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

// scenarioFiles lists the scenarios, filtered by MEDUSA_HTTP_SCENARIO.
func scenarioFiles() []string {
	files, _ := filepath.Glob("scenarios/*.json")
	if only := os.Getenv("MEDUSA_HTTP_SCENARIO"); only != "" {
		var out []string
		for _, f := range files {
			if strings.TrimSuffix(filepath.Base(f), ".json") == only {
				out = append(out, f)
			}
		}
		return out
	}
	return files
}

// goServer starts the Go application on a fresh database holding seed.sql.
func goServer(t *testing.T, scenario string) string {
	t.Helper()
	if base := os.Getenv("MEDUSA_HTTP_TARGET_URL"); base != "" {
		return base
	}
	dbURL := setupDatabase(t, "medusa_http_go_"+strings.ReplaceAll(scenario, "-", "_"))
	// Same CORS settings as record.sh passes to upstream.
	cfg := config.DefineConfig(config.InputConfig{ProjectConfig: config.ProjectConfig{
		DatabaseURL: dbURL,
		HTTP:        config.HttpConfig{AdminCors: "http://localhost:5173", AuthCors: "http://localhost:5173", StoreCors: "http://localhost:8000"},
	}})
	app, err := loaders.Initialize(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(app.Handler)
	t.Cleanup(func() {
		srv.Close()
		app.Shutdown()
	})
	return srv.URL
}

// TestMatchUpstream runs every scenario against the Go application (each on
// a fresh database, or against MEDUSA_HTTP_TARGET_URL) and compares every
// response (status, headers, cookies, body with its key order) with the
// golden responses recorded from upstream by record.sh.
func TestMatchUpstream(t *testing.T) {
	for _, file := range scenarioFiles() {
		name := strings.TrimSuffix(filepath.Base(file), ".json")
		t.Run(name, func(t *testing.T) {
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
			seedSQL(t)
			base := goServer(t, name)
			results := runScenario(t, base, steps)
			wantP, gotP := newPlaceholders(), newPlaceholders()
			for _, s := range steps {
				want, _ := golden.Get(s.Name)
				w := pretty(wantP.normalize(sortUnordered(want, s.Unordered)))
				g := pretty(gotP.normalize(sortUnordered(asRecord(results[s.Name]), s.Unordered)))
				if w != g {
					t.Errorf("%s: mismatch\n--- want\n%s\n--- got\n%s", s.Name, w, g)
				}
			}
		})
	}
}
