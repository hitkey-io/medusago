package http

import (
	"context"
	nethttp "net/http"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// MedusaContainer is the request scope routes resolve dependencies from.
type MedusaContainer interface {
	Resolve(key string) any
}

// QueryConfig mirrors req.queryConfig / req.remoteQueryConfig.
type QueryConfig struct {
	Entity     string
	Fields     []string
	Pagination *dal.Record
	// WithDeleted is nil when undefined.
	WithDeleted *bool
}

// ListConfig mirrors req.listConfig / req.retrieveConfig.
type ListConfig struct {
	Select      []string
	Relations   []string
	Skip        *float64
	Take        *float64
	Order       *dal.Record
	WithDeleted *bool
}

// MedusaRequest mirrors MedusaRequest / AuthenticatedMedusaRequest.
type MedusaRequest struct {
	*nethttp.Request
	Params map[string]string
	// Query is the parsed query string (qs, extended).
	Query *dal.Record
	// Body is the parsed request body: an ordered JSON value, a string
	// (text/plain) or a *dal.Record (urlencoded). Nil when not parsed.
	Body    any
	RawBody []byte
	Scope   MedusaContainer
	Session *Session
	// AuthContext is the verified auth context (JWT payload or session).
	AuthContext *dal.Record

	ValidatedBody  any
	ValidatedQuery *dal.Record
	QueryConfig    QueryConfig
	// ListConfig / RemoteQueryConfig mirror req.listConfig and
	// req.remoteQueryConfig.
	ListConfig        *ListConfig
	RetrieveConfig    *ListConfig
	RemoteQueryConfig QueryConfig
	FilterableFields  *dal.Record
	Allowed           []string
	// Disallowed mirrors req.disallowed (nil = undefined).
	Disallowed              []string
	RestrictedFields        []string
	StoreRelationsLimit     int
	PublishableKeyContext   *dal.Record
	Locale                  string
	AdditionalDataValidator any
	// Errors mirrors req.errors (wrapHandler answers 400 when set).
	Errors []any
	// Locals holds values shared between middlewares.
	Locals map[string]any
	// Files mirrors req.files set by multer (nil = undefined).
	Files []UploadedFile
}

// UploadedFile mirrors Express.Multer.File (memory storage).
type UploadedFile struct {
	FieldName    string
	OriginalName string
	MimeType     string
	Buffer       []byte
}

// NewMedusaRequest wraps r.
func NewMedusaRequest(r *nethttp.Request) *MedusaRequest {
	return &MedusaRequest{Request: r, Params: map[string]string{}, Locals: map[string]any{}}
}

// Ctx returns the request context.
func (r *MedusaRequest) Ctx() context.Context { return r.Request.Context() }

// Path mirrors req.path: the URL path (not decoded).
func (r *MedusaRequest) Path() string { return r.URL.EscapedPath() }

// AuthString returns a string field of the auth context.
func (r *MedusaRequest) AuthString(key string) string {
	if r.AuthContext == nil {
		return ""
	}
	v, _ := r.AuthContext.Get(key)
	s, _ := v.(string)
	return s
}

// Protocol mirrors req.protocol with "trust proxy" 1.
func (r *MedusaRequest) Protocol() string {
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		if i := strings.Index(p, ","); i >= 0 {
			p = p[:i]
		}
		return strings.TrimSpace(p)
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

// MedusaResponse mirrors the Express response.
type MedusaResponse struct {
	w          nethttp.ResponseWriter
	req        *MedusaRequest
	status     int
	headerSent bool
	finished   bool
	// beforeHeaders run once right before the headers are written
	// (express-session sets its cookie there).
	beforeHeaders []func()
	// afterEnd run once the response is complete.
	afterEnd []func()
}

func newResponse(w nethttp.ResponseWriter, req *MedusaRequest) *MedusaResponse {
	return &MedusaResponse{w: w, req: req, status: 200}
}

// Header returns the response headers.
func (r *MedusaResponse) Header() nethttp.Header { return r.w.Header() }

// SetHeader mirrors res.set.
func (r *MedusaResponse) SetHeader(k, v string) { r.w.Header().Set(k, v) }

// HeadersSent mirrors res.headersSent.
func (r *MedusaResponse) HeadersSent() bool { return r.headerSent }

// Finished reports whether the response has been sent.
func (r *MedusaResponse) Finished() bool { return r.finished }

// Status mirrors res.status.
func (r *MedusaResponse) Status(code int) *MedusaResponse {
	r.status = code
	return r
}

// StatusCode returns the status set so far.
func (r *MedusaResponse) StatusCode() int { return r.status }

// OnHeaders registers a hook run before the headers are written.
func (r *MedusaResponse) OnHeaders(fn func()) { r.beforeHeaders = append(r.beforeHeaders, fn) }

// OnEnd registers a hook run once the response is sent.
func (r *MedusaResponse) OnEnd(fn func()) { r.afterEnd = append(r.afterEnd, fn) }

// Writer returns the underlying writer, for streaming responses. Headers
// are flushed (hooks run) on first use.
func (r *MedusaResponse) Writer() nethttp.ResponseWriter {
	r.writeHeader(r.status)
	return r.w
}

// Handoff runs the header hooks and returns the underlying writer without
// writing the status: the caller (e.g. http.ServeContent) writes it.
func (r *MedusaResponse) Handoff() nethttp.ResponseWriter {
	if !r.headerSent {
		hooks := r.beforeHeaders
		r.beforeHeaders = nil
		for i := len(hooks) - 1; i >= 0; i-- {
			hooks[i]()
		}
		r.headerSent = true
	}
	r.finished = true
	return r.w
}

func (r *MedusaResponse) writeHeader(code int) {
	if r.headerSent {
		return
	}
	hooks := r.beforeHeaders
	r.beforeHeaders = nil
	for i := len(hooks) - 1; i >= 0; i-- {
		hooks[i]()
	}
	r.headerSent = true
	r.w.WriteHeader(code)
}

func (r *MedusaResponse) writeRaw(code int, body []byte) {
	if r.finished {
		return
	}
	r.status = code
	r.writeHeader(code)
	if len(body) > 0 && r.req.Method != "HEAD" {
		_, _ = r.w.Write(body)
	}
	r.end()
}

func (r *MedusaResponse) end() {
	r.finished = true
	hooks := r.afterEnd
	r.afterEnd = nil
	for _, h := range hooks {
		h()
	}
}

// End mirrors res.end().
func (r *MedusaResponse) End() {
	if r.finished {
		return
	}
	r.writeHeader(r.status)
	r.end()
}

// JSON mirrors res.json: JSON.stringify then res.send.
func (r *MedusaResponse) JSON(v any) error {
	raw, err := dal.MarshalValue(v)
	if err != nil {
		return err
	}
	if r.w.Header().Get("Content-Type") == "" {
		r.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	r.sendBytes(raw)
	return nil
}

// Send mirrors res.send(string): text/html by default.
func (r *MedusaResponse) Send(body string) {
	if r.w.Header().Get("Content-Type") == "" {
		r.w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	r.sendBytes([]byte(body))
}

// SendStatus mirrors res.sendStatus.
func (r *MedusaResponse) SendStatus(code int) {
	r.status = code
	r.Header().Set("Content-Type", "text/plain; charset=utf-8")
	r.sendBytes([]byte(nethttp.StatusText(code)))
}

// sendBytes mirrors the tail of res.send: ETag, freshness, 204/304.
func (r *MedusaResponse) sendBytes(body []byte) {
	if r.finished {
		return
	}
	h := r.w.Header()
	if h.Get("ETag") == "" {
		h.Set("ETag", weakETag(body))
	}
	status := r.status
	if r.isFresh() {
		status = 304
	}
	if status == 204 || status == 304 {
		h.Del("Content-Type")
		h.Del("Content-Length")
		h.Del("Transfer-Encoding")
		body = nil
	} else {
		h.Set("Content-Length", itoa(len(body)))
	}
	r.writeRaw(status, body)
}

// isFresh mirrors req.fresh (etag only).
func (r *MedusaResponse) isFresh() bool {
	m := r.req.Method
	if m != "GET" && m != "HEAD" {
		return false
	}
	if !((r.status >= 200 && r.status < 300) || r.status == 304) {
		return false
	}
	inm := r.req.Header.Get("If-None-Match")
	if inm == "" {
		return false
	}
	if cc := r.req.Header.Get("Cache-Control"); strings.Contains(cc, "no-cache") {
		return false
	}
	if inm == "*" {
		return true
	}
	etag := r.w.Header().Get("ETag")
	for _, t := range strings.Split(inm, ",") {
		t = strings.TrimSpace(t)
		if t == etag || t == "W/"+etag || "W/"+t == etag {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
