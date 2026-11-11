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
	mfautils "github.com/hitkey-io/medusago/packages/modules/auth/src/utils"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/integration-tests/http-diff/testapp/brand"
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
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
	// SortBy maps body keys whose arrays upstream returns in a
	// nondeterministic order (e.g. by generated id) to the item field they
	// are compared sorted by.
	SortBy map[string]string `json:"sortby,omitempty"`
	// Ignore lists dotted body paths left out of the comparison: values that
	// upstream computes nondeterministically (concurrent writes on one
	// entity manager).
	Ignore []string `json:"ignore,omitempty"`
	// Files sends a multipart/form-data body with these files.
	Files []StepFile `json:"files,omitempty"`
	// Until repeats the request (for up to 30s) until this dotted body path
	// resolves to a value ("path=value": to that value): it waits for
	// background workflows.
	Until string `json:"until,omitempty"`
	// UntilAbsent repeats the request (for up to 30s) while this dotted
	// body path resolves to a value: it waits for asynchronous subscribers
	// (index engine updates) to remove something.
	UntilAbsent string `json:"until_absent,omitempty"`
	// Totp names a body path ("step.path") holding a TOTP secret: the
	// current code replaces "__TOTP__" in the body.
	Totp string `json:"totp,omitempty"`
	// Parallel groups consecutive steps sent concurrently (e.g. two
	// completions of one cart); they may only reference earlier steps.
	Parallel string `json:"parallel,omitempty"`
	// DelayMs waits before sending the step (staggers a parallel group).
	DelayMs int `json:"delay_ms,omitempty"`
	// Service sends the request to a fake third-party service of the
	// scenario (e.g. "stripe") instead of the server.
	Service string `json:"service,omitempty"`
	// Sign signs the body like the named provider's webhooks ("stripe":
	// a stripe-signature header).
	Sign string `json:"sign,omitempty"`
	// External requests an absolute URL (a fake service, an uploaded file)
	// as is instead of a path of the server.
	External string `json:"external,omitempty"`
}

var otpSecretRe = regexp.MustCompile(`secret=[A-Z2-7]+`)

func maskRandom(v any) any {
	switch t := v.(type) {
	case string:
		return "<random>"
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = "<random>"
		}
		return out
	}
	return v
}

// lookupPath resolves a dotted path ("notifications.0.data") in a body.
func lookupPath(v any, path string) any {
	for _, part := range strings.Split(path, ".") {
		switch t := v.(type) {
		case *dal.Record:
			v, _ = t.Get(part)
		case []any:
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(t) {
				return nil
			}
			v = t[i]
		default:
			return nil
		}
	}
	return v
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

// dropIgnored removes the listed dotted paths from the body.
func dropIgnored(rec any, paths []string) any {
	r, ok := rec.(*dal.Record)
	if !ok || len(paths) == 0 {
		return rec
	}
	for _, path := range paths {
		dropPath(r, strings.Split("body."+path, "."))
	}
	return rec
}

// dropPath removes a dotted path; "*" (or an index) descends into every
// (or one) element of an array.
func dropPath(v any, parts []string) {
	switch t := v.(type) {
	case *dal.Record:
		if len(parts) == 1 {
			t.Delete(parts[0])
			return
		}
		next, _ := t.Get(parts[0])
		dropPath(next, parts[1:])
	case []any:
		for i, e := range t {
			if parts[0] == "*" || parts[0] == strconv.Itoa(i) {
				if len(parts) > 1 {
					dropPath(e, parts[1:])
				}
			}
		}
	}
}

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

// sortByField sorts the listed body arrays by the string form of an item
// field.
func sortByField(rec any, keys map[string]string) any {
	r, ok := rec.(*dal.Record)
	if !ok || len(keys) == 0 {
		return rec
	}
	body, _ := r.Get("body")
	b, ok := body.(*dal.Record)
	if !ok {
		return rec
	}
	for k, field := range keys {
		v, _ := b.Get(k)
		arr, ok := v.([]any)
		if !ok {
			continue
		}
		sorted := append([]any{}, arr...)
		key := func(item any) string {
			if ir, ok := item.(*dal.Record); ok {
				fv, _ := ir.Get(field)
				return fmt.Sprint(fv)
			}
			return ""
		}
		sort.SliceStable(sorted, func(i, j int) bool { return key(sorted[i]) < key(sorted[j]) })
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
	var tokensMu sync.Mutex

	bodies := map[string]any{}
	// doStep sends one step (bodies is only read: the steps of a parallel
	// group are stored once the group is done).
	doStep := func(step Step) (Response, error) {
		step.Path = resolveRefs(step.Path, bodies, true)
		// absolute URLs from bodies (file urls) are requested on the target
		if strings.HasPrefix(step.Path, "http%3A") {
			step.Path, _ = url.QueryUnescape(step.Path)
		}
		if u, err := url.Parse(step.Path); err == nil && u.IsAbs() {
			step.Path = u.RequestURI()
		}
		if step.DelayMs > 0 {
			time.Sleep(time.Duration(step.DelayMs) * time.Millisecond)
		}
		deadline := time.Now().Add(30 * time.Second)
		for {
			var body io.Reader
			if len(step.Body) > 0 {
				raw := resolveRefs(string(step.Body), bodies, false)
				if step.Totp != "" {
					stepName, path, _ := strings.Cut(step.Totp, ".")
					secret := fmt.Sprint(lookupPath(bodies[stepName], path))
					code, err := mfautils.GenerateTotpCode(secret, 6, 30, time.Now())
					if err != nil {
						return Response{}, fmt.Errorf("%s: totp: %v", step.Name, err)
					}
					raw = strings.ReplaceAll(raw, "__TOTP__", code)
				}
				body = strings.NewReader(raw)
			}
			contentType := ""
			if len(step.Body) > 0 {
				contentType = "application/json"
			}
			if len(step.Files) > 0 {
				body, contentType = multipartBody(step.Files)
			}
			base := baseURL
			if step.Service != "" {
				base = serviceURLs[step.Service]
			}
			headers := map[string]string{}
			for k, v := range step.Headers {
				headers[k] = v
			}
			if step.Sign != "" && len(step.Body) > 0 {
				raw, _ := io.ReadAll(body)
				signed, signHeaders := signBody(step.Sign, string(raw))
				body = strings.NewReader(signed)
				for k, v := range signHeaders {
					headers[k] = v
				}
			}
			target := base + step.Path
			if step.External != "" {
				target = resolveTextRefs(step.External, bodies)
			}
			req, err := http.NewRequest(step.Method, target, body)
			if err != nil {
				return Response{}, err
			}
			if contentType != "" {
				req.Header.Set("Content-Type", contentType)
			}
			if step.Origin != "" {
				req.Header.Set("Origin", step.Origin)
			}
			for k, v := range headers {
				req.Header.Set(k, resolveRefs(v, bodies, true))
			}
			if step.Bearer != "" {
				tokensMu.Lock()
				req.Header.Set("Authorization", "Bearer "+tokens[step.Bearer])
				tokensMu.Unlock()
			}
			c := client
			if step.NoCookies {
				c = plain
			}
			res, err := c.Do(req)
			if err != nil {
				return Response{}, fmt.Errorf("%s: %v", step.Name, err)
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
							tokensMu.Lock()
							tokens[step.Name] = fmt.Sprint(tok)
							tokensMu.Unlock()
						}
					}
				} else {
					rec.Body = string(raw)
				}
			}
			if step.Until != "" && !untilReached(rec.Body, step.Until) && time.Now().Before(deadline) {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			if step.UntilAbsent != "" && lookupPath(rec.Body, step.UntilAbsent) != nil && time.Now().Before(deadline) {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			return rec, nil
		}
	}

	for i := 0; i < len(steps); {
		// consecutive steps of one parallel group are sent concurrently
		j := i + 1
		if steps[i].Parallel != "" {
			for j < len(steps) && steps[j].Parallel == steps[i].Parallel {
				j++
			}
		}
		group := steps[i:j]
		results := make([]Response, len(group))
		errs := make([]error, len(group))
		var wg sync.WaitGroup
		for k := range group {
			wg.Add(1)
			go func(k int) {
				defer wg.Done()
				results[k], errs[k] = doStep(group[k])
			}(k)
		}
		wg.Wait()
		for k, step := range group {
			if errs[k] != nil {
				t.Fatal(errs[k])
			}
			out[step.Name] = results[k]
			bodies[step.Name] = results[k].Body
		}
		i = j
	}
	return out
}

// untilReached reports whether the body satisfies an "until" condition.
func untilReached(body any, cond string) bool {
	path, want, hasValue := strings.Cut(cond, "=")
	v := lookupPath(body, path)
	if !hasValue {
		return v != nil
	}
	return v != nil && fmt.Sprint(v) == want
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
		cur := refValue(bodies, sm[1], sm[2])
		if inPath {
			return url.QueryEscape(fmt.Sprint(cur))
		}
		raw, _ := dal.MarshalValue(cur)
		return string(raw)
	})
}

// resolveTextRefs replaces {{step.path}} with the plain text of the value.
func resolveTextRefs(s string, bodies map[string]any) string {
	return refRe.ReplaceAllStringFunc(s, func(m string) string {
		sm := refRe.FindStringSubmatch(m)
		return fmt.Sprint(refValue(bodies, sm[1], sm[2]))
	})
}

// refValue resolves a reference path in a step body: object keys, array
// indexes, and "?name" for a query parameter of a URL string.
func refValue(bodies map[string]any, stepName, path string) any {
	var cur any = bodies[stepName]
	{
		for _, seg := range strings.Split(path, ".") {
			if name, ok := strings.CutPrefix(seg, "?"); ok {
				if s, isStr := cur.(string); isStr {
					if u, err := url.Parse(s); err == nil {
						cur = u.Query().Get(name)
						continue
					}
				}
				cur = nil
				continue
			}
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
	}
	return cur
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
		t.Run(strings.TrimSuffix(filepath.Base(file), ".json"), func(t *testing.T) { recordScenario(t, base, file) })
	}
}

// recordScenario records one scenario (with its fake services).
func recordScenario(t *testing.T, base, file string) {
	{
		steps := loadScenario(t, file)
		env := scenarioEnv(t, strings.TrimSuffix(filepath.Base(file), ".json"))
		for k, v := range env {
			t.Setenv(k, v)
		}
		startFakes(t, env)
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

var (
	embeddedIDRe = regexp.MustCompile(`[a-z]+_[0-9A-HJKMNP-TV-Z]{26}`)
	// Date.prototype.toString: the zone name is ICU's, in the system locale
	jsDateZoneRe   = regexp.MustCompile(`(GMT[+-]\d{4}) \([^)]*\)`)
	embeddedDateRe = regexp.MustCompile(`\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d{3})?Z`)
)

var (
	// presigned S3 URLs: signing time and signature
	amzDateRe       = regexp.MustCompile(`X-Amz-Date=\d{8}T\d{6}Z`)
	amzSignatureRe  = regexp.MustCompile(`X-Amz-Signature=[0-9a-f]{64}`)
	amzCredentialRe = regexp.MustCompile(`(X-Amz-Credential=[^&%]+%2F)\d{8}%2F`)
	// OAuth state keys (crypto.randomBytes(32).toString("hex"))
	oauthStateRe = regexp.MustCompile(`state=[0-9a-f]{64}\b`)
	// ULIDs inside file keys ("name-<ULID>.ext", "<ULID>.png")
	keyULIDRe = regexp.MustCompile(`(^|[-/])([0-9A-HJKMNP-TV-Z]{26})([./?%]|$)`)
)

// maskProviderValues masks the generated parts of values produced by the
// module providers (S3 keys and presigned URLs, OAuth redirects).
func (p *placeholders) maskProviderValues(s string) string {
	if strings.Contains(s, "X-Amz-") {
		s = amzDateRe.ReplaceAllString(s, "X-Amz-Date=<date>")
		s = amzSignatureRe.ReplaceAllString(s, "X-Amz-Signature=<signature>")
		s = amzCredentialRe.ReplaceAllString(s, "${1}<date>%2F")
	}
	if strings.Contains(s, "state=") {
		s = oauthStateRe.ReplaceAllString(s, "state=<state>")
	}
	return keyULIDRe.ReplaceAllStringFunc(s, func(m string) string {
		sm := keyULIDRe.FindStringSubmatch(m)
		return sm[1] + p.of(sm[2]) + sm[3]
	})
}

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
	if !gen && kind == "" {
		s = p.maskProviderValues(s)
	}
	if !gen {
		if kind == "" && len(s) > 30 {
			// generated ids and dates inside messages (and CSV files)
			s = embeddedIDRe.ReplaceAllStringFunc(s, func(id string) string {
				if g, k := isGenerated(id); g && k == "id" {
					return p.of(id)
				}
				return id
			})
			s = jsDateZoneRe.ReplaceAllString(s, "$1 (<zone>)")
			return embeddedDateRe.ReplaceAllStringFunc(s, func(d string) string {
				if g, k := isGenerated(d); g && k == "date" {
					return "<date>"
				}
				return d
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
			case k == "secret" || k == "recovery_codes" || k == "token_hash":
				// random TOTP secrets and recovery codes
				out.Set(k, maskRandom(x))
			case k == "otpauth_url":
				out.Set(k, otpSecretRe.ReplaceAllString(fmt.Sprint(x), "secret=<secret>"))
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

// scenarioFiles lists the scenarios, filtered by MEDUSA_HTTP_SCENARIO (a
// comma separated list of names).
func scenarioFiles() []string {
	files, _ := filepath.Glob("scenarios/*.json")
	if only := os.Getenv("MEDUSA_HTTP_SCENARIO"); only != "" {
		names := map[string]bool{}
		for _, n := range strings.Split(only, ",") {
			names[strings.TrimSpace(n)] = true
		}
		var out []string
		for _, f := range files {
			if names[strings.TrimSuffix(filepath.Base(f), ".json")] {
				out = append(out, f)
			}
		}
		return out
	}
	return files
}

// goServer starts the Go application on a fresh database holding seed.sql.
// mfaEncryptionKey is the AUTH_MFA_ENCRYPTION_KEY both servers run with
// (record.sh passes the same value to upstream).
const mfaEncryptionKey = "http-diff-mfa-encryption-key"

// scenarioEnv reads scenarios/<name>.env (KEY=VALUE lines, e.g. feature
// flags): both servers run with these variables (record.sh exports them).
func scenarioEnv(t *testing.T, scenario string) map[string]string {
	t.Helper()
	out := map[string]string{}
	data, err := os.ReadFile(filepath.Join("scenarios", scenario+".env"))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			// TEMPLATE_DB only selects record.sh's template database.
			if strings.TrimSpace(k) == "TEMPLATE_DB" {
				continue
			}
			out[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	return out
}

func goServer(t *testing.T, scenario string) string {
	t.Helper()
	t.Setenv("AUTH_MFA_ENCRYPTION_KEY", mfaEncryptionKey)
	for k, v := range scenarioEnv(t, scenario) {
		t.Setenv(k, v)
	}
	// scenarios of the Redis providers need the Redis their .env names
	if redisURL := os.Getenv("MEDUSA_TEST_REDIS_URL"); redisURL != "" {
		if u, err := url.Parse(redisURL); err == nil {
			conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second)
			if err != nil {
				t.Skipf("Redis %s is not reachable (start it, e.g. docker run -p %s:6379 redis:7-alpine)", redisURL, u.Port())
			}
			conn.Close()
		}
	}
	if base := os.Getenv("MEDUSA_HTTP_TARGET_URL"); base != "" {
		return base
	}
	projectMiddlewares := registerTestSearchIndexes()
	prepareRedis(t)
	if endpoint := os.Getenv("MEDUSA_SESSION_DYNAMODB_ENDPOINT"); endpoint != "" {
		// the scenario needs a local DynamoDB (docker amazon/dynamodb-local)
		if u, err := url.Parse(endpoint); err == nil {
			if conn, err := net.DialTimeout("tcp", u.Host, 2*time.Second); err != nil {
				t.Skipf("DynamoDB at %s is not reachable: %v", endpoint, err)
			} else {
				conn.Close()
			}
		}
	}
	// The links of the brand test module (testapp/src/links) are defined
	// for the scenario only.
	var routes []medusahttp.RouteFile
	if os.Getenv("MEDUSA_TEST_BRAND") == "true" {
		utilsmodulessdk.ClearCustomLinks()
		t.Cleanup(utilsmodulessdk.ClearCustomLinks)
		if err := brand.DefineLinks(); err != nil {
			t.Fatal(err)
		}
		routes = brand.Routes
	}
	dbURL := setupDatabase(t, "medusa_http_go_"+strings.ReplaceAll(scenario, "-", "_"))
	// Same CORS settings as record.sh passes to upstream.
	cfg := config.DefineConfig(config.InputConfig{ProjectConfig: config.ProjectConfig{
		DatabaseURL:    dbURL,
		HTTP:           config.HttpConfig{AdminCors: "http://localhost:5173", AuthCors: "http://localhost:5173", StoreCors: "http://localhost:8000"},
		SessionOptions: scenarioSessionOptions(),
	}, Modules: scenarioModules()})
	app, err := loaders.InitializeWithOptions(context.Background(), cfg, loaders.Options{Middlewares: projectMiddlewares, Routes: routes, OnLoad: append(scenarioHooks(t), registerEventsCapture)})
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

// scenarioSessionOptions mirrors the sessionOptions of the reference app
// (medusa-config.js of REF_APP): the DynamoDB session store when
// MEDUSA_SESSION_DYNAMODB_ENDPOINT is set (a local DynamoDB).
func scenarioSessionOptions() *config.SessionOptions {
	endpoint := os.Getenv("MEDUSA_SESSION_DYNAMODB_ENDPOINT")
	if endpoint == "" {
		return nil
	}
	table := os.Getenv("MEDUSA_SESSION_DYNAMODB_TABLE")
	if table == "" {
		table = "sessions"
	}
	prefix := os.Getenv("MEDUSA_SESSION_DYNAMODB_PREFIX")
	if prefix == "" {
		prefix = "sess:"
	}
	return &config.SessionOptions{DynamodbOptions: map[string]any{
		"table":  table,
		"prefix": prefix,
		"clientOptions": map[string]any{
			"endpoint":    endpoint,
			"region":      "us-east-1",
			"credentials": map[string]any{"accessKeyId": "local", "secretAccessKey": "local"},
		},
	}}
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
			startFakes(t, scenarioEnv(t, name))
			base := goServer(t, name)
			results := runScenario(t, base, steps)
			wantP, gotP := newPlaceholders(), newPlaceholders()
			for _, s := range steps {
				want, _ := golden.Get(s.Name)
				w := pretty(wantP.normalize(sortByField(sortUnordered(dropIgnored(want, s.Ignore), s.Unordered), s.SortBy)))
				g := pretty(gotP.normalize(sortByField(sortUnordered(dropIgnored(asRecord(results[s.Name]), s.Ignore), s.Unordered), s.SortBy)))
				if w != g {
					t.Errorf("%s: mismatch\n--- want\n%s\n--- got\n%s", s.Name, w, g)
				}
			}
		})
	}
}
