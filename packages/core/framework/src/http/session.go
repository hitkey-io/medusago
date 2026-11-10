package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// This file reproduces express-session 1.19 (cookie-signature, uid-safe ids,
// Session/Cookie serialization, MemoryStore) as configured by the express
// loader, so that sessions behave and serialize exactly like upstream and a
// session store (connect-redis) can be shared with Node servers.

// SessionCookieOptions mirrors the express-session cookie options.
type SessionCookieOptions struct {
	Path     string
	Domain   string
	HttpOnly bool
	Secure   bool
	// SameSite: "", "lax", "strict", "none", "true" ("" = false, "true" =
	// the boolean true, serialized as SameSite=Strict).
	SameSite    string
	Partitioned bool
	// Priority: "", "low", "medium", "high".
	Priority string
	// MaxAge in milliseconds; 0 = browser session cookie.
	MaxAge int64
}

// SessionOptions mirrors the options passed to express-session.
type SessionOptions struct {
	Name              string
	Secret            string
	Resave            bool
	Rolling           bool
	SaveUninitialized bool
	Cookie            SessionCookieOptions
	Store             SessionStore
}

// SessionStore mirrors the express-session store interface. Values are the
// JSON representation of the session (JSON.stringify(req.session), the shape
// connect-redis stores); expires is session.cookie.expires (zero = null).
type SessionStore interface {
	Get(sid string) ([]byte, bool, error)
	Set(sid string, data []byte, expires time.Time) error
	Destroy(sid string) error
	Touch(sid string, data []byte, expires time.Time) error
}

// Session mirrors req.session.
type Session struct {
	ID     string
	cookie *sessionCookie
	opts   *SessionOptions
	// Data holds the session values (everything but "cookie").
	Data      *dal.Record
	destroyed bool
	store     SessionStore
}

// Get returns a session value.
func (s *Session) Get(key string) (any, bool) {
	if s == nil || s.Data == nil {
		return nil, false
	}
	return s.Data.Get(key)
}

// Set sets a session value.
func (s *Session) Set(key string, v any) { s.Data.Set(key, v) }

// Destroy mirrors req.session.destroy.
func (s *Session) Destroy() error {
	s.destroyed = true
	return s.store.Destroy(s.ID)
}

// Expires returns session.cookie.expires (zero when null).
func (s *Session) Expires() time.Time {
	if s.cookie == nil || s.cookie.expires == nil {
		return time.Time{}
	}
	return *s.cookie.expires
}

// sessionCookie mirrors express-session's Cookie. attrs holds the defined
// (not undefined) attributes among cookieDataKeys, with their JS values.
type sessionCookie struct {
	originalMaxAge any // float64 or nil (null)
	expires        *time.Time
	attrs          map[string]any
}

// cookieAttrKeys are the keys of Cookie.data besides originalMaxAge and
// expires, in getter order.
var cookieAttrKeys = []string{"partitioned", "priority", "secure", "httpOnly", "domain", "path", "sameSite"}

// newSessionCookie mirrors `new Cookie(cookieOptions)` for a generated
// session.
func newSessionCookie(o SessionCookieOptions) *sessionCookie {
	c := &sessionCookie{attrs: map[string]any{
		"path":     o.Path,
		"httpOnly": o.HttpOnly,
		"secure":   o.Secure,
		"sameSite": sameSiteJS(o.SameSite),
	}}
	if c.attrs["path"] == "" {
		c.attrs["path"] = "/"
	}
	if o.Domain != "" {
		c.attrs["domain"] = o.Domain
	}
	if o.Partitioned {
		c.attrs["partitioned"] = true
	}
	if o.Priority != "" {
		c.attrs["priority"] = o.Priority
	}
	if o.MaxAge > 0 {
		c.originalMaxAge = float64(o.MaxAge)
		c.touch()
	}
	return c
}

func sameSiteJS(v string) any {
	switch v {
	case "":
		return false
	case "true":
		return true
	}
	return v
}

// parseSessionCookie mirrors Store.createSession: the stored cookie object
// is revived with `new Cookie(sess.cookie)`, expires becomes a Date and
// originalMaxAge is kept intact.
func parseSessionCookie(v any) *sessionCookie {
	c := &sessionCookie{attrs: map[string]any{"path": "/", "httpOnly": true}}
	rec, ok := v.(*dal.Record)
	if !ok {
		return c
	}
	for _, k := range cookieAttrKeys {
		if x, ok := rec.Get(k); ok {
			c.attrs[k] = x
		}
	}
	if x, ok := rec.Get("originalMaxAge"); ok {
		if n, ok := x.(float64); ok {
			c.originalMaxAge = n
		}
	}
	if x, ok := rec.Get("expires"); ok {
		if es, ok := x.(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, es); err == nil {
				c.expires = &t
			}
		}
	}
	return c
}

// touch mirrors session.touch() (resetMaxAge): cookie.maxAge =
// cookie.originalMaxAge, i.e. expires = now + originalMaxAge.
func (c *sessionCookie) touch() {
	ms, ok := c.originalMaxAge.(float64)
	if !ok {
		c.expires = nil
		return
	}
	t := time.UnixMilli(time.Now().UnixMilli() + int64(ms)).UTC()
	c.expires = &t
}

// data mirrors Cookie.data / toJSON (undefined attributes are omitted).
func (c *sessionCookie) data() *dal.Record {
	r := dal.NewRecord()
	r.Set("originalMaxAge", c.originalMaxAge)
	for _, k := range cookieAttrKeys[:2] {
		if v, ok := c.attrs[k]; ok {
			r.Set(k, v)
		}
	}
	if c.expires != nil {
		r.Set("expires", *c.expires)
	} else {
		r.Set("expires", nil)
	}
	for _, k := range cookieAttrKeys[2:] {
		if v, ok := c.attrs[k]; ok {
			r.Set(k, v)
		}
	}
	return r
}

func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case string:
		return x != ""
	case float64:
		return x != 0
	}
	return true
}

// serializeOptions mirrors the options cookie.serialize receives
// (req.session.cookie.data).
func (c *sessionCookie) serializeOptions() SessionCookieOptions {
	var o SessionCookieOptions
	o.Path, _ = c.attrs["path"].(string)
	o.Domain, _ = c.attrs["domain"].(string)
	o.HttpOnly = truthy(c.attrs["httpOnly"])
	o.Secure = truthy(c.attrs["secure"])
	o.Partitioned = truthy(c.attrs["partitioned"])
	o.Priority, _ = c.attrs["priority"].(string)
	switch v := c.attrs["sameSite"].(type) {
	case bool:
		if v {
			o.SameSite = "true"
		}
	case string:
		o.SameSite = v
	}
	return o
}

// serialize mirrors JSON.stringify(req.session).
func (s *Session) serialize() []byte {
	r := dal.NewRecord()
	r.Set("cookie", s.cookie.data())
	for _, k := range s.Data.Keys() {
		v, _ := s.Data.Get(k)
		r.Set(k, v)
	}
	raw, _ := r.MarshalJSON()
	return raw
}

// hash mirrors express-session hash(): the session without its cookie.
func (s *Session) hash() string {
	raw, _ := s.Data.MarshalJSON()
	sum := sha256.Sum256(raw)
	return string(sum[:])
}

// MemoryStore mirrors express-session's MemoryStore.
type MemoryStore struct {
	mu       sync.Mutex
	sessions map[string]memoryEntry
}

type memoryEntry struct {
	data    []byte
	expires time.Time
}

// NewMemoryStore creates an empty store.
func NewMemoryStore() *MemoryStore { return &MemoryStore{sessions: map[string]memoryEntry{}} }

func (m *MemoryStore) Get(sid string) ([]byte, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.sessions[sid]
	if !ok {
		return nil, false, nil
	}
	if !e.expires.IsZero() && !e.expires.After(time.Now()) {
		delete(m.sessions, sid)
		return nil, false, nil
	}
	return e.data, true, nil
}

func (m *MemoryStore) Set(sid string, data []byte, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[sid] = memoryEntry{data: data, expires: expires}
	return nil
}

func (m *MemoryStore) Destroy(sid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, sid)
	return nil
}

// Touch mirrors MemoryStore.touch: only the stored cookie is replaced.
func (m *MemoryStore) Touch(sid string, data []byte, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.sessions[sid]
	if !ok {
		return nil
	}
	cur, err1 := dal.DecodeOrderedJSON(e.data)
	next, err2 := dal.DecodeOrderedJSON(data)
	if cr, ok := cur.(*dal.Record); ok && err1 == nil && err2 == nil {
		if nr, ok := next.(*dal.Record); ok {
			c, _ := nr.Get("cookie")
			cr.Set("cookie", c)
			if raw, err := cr.MarshalJSON(); err == nil {
				e.data = raw
			}
		}
	}
	e.expires = expires
	m.sessions[sid] = e
	return nil
}

// signCookie mirrors cookie-signature sign.
func signCookie(val, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(val))
	sig := strings.TrimRight(base64.StdEncoding.EncodeToString(mac.Sum(nil)), "=")
	return val + "." + sig
}

// unsignCookie mirrors cookie-signature unsign.
func unsignCookie(input, secret string) (string, bool) {
	i := strings.LastIndex(input, ".")
	if i < 0 {
		return "", false
	}
	val := input[:i]
	expected := signCookie(val, secret)
	if hmac.Equal([]byte(expected), []byte(input)) {
		return val, true
	}
	return "", false
}

// uidSafe mirrors uid-safe(24): base64url of 24 random bytes.
func uidSafe() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// encodeURIComponent mirrors the JS function.
func encodeURIComponent(s string) string {
	out := url.QueryEscape(s)
	out = strings.ReplaceAll(out, "+", "%20")
	for _, c := range []string{"!", "'", "(", ")", "*"} {
		out = strings.ReplaceAll(out, url.QueryEscape(c), c)
	}
	return out
}

// SerializeCookie mirrors cookie.serialize (cookie 0.7).
func SerializeCookie(name, value string, o SessionCookieOptions, expires *time.Time, maxAge *int64) string {
	var b strings.Builder
	b.WriteString(name + "=" + encodeURIComponent(value))
	if maxAge != nil {
		b.WriteString("; Max-Age=" + itoa(int(*maxAge)))
	}
	if o.Domain != "" {
		b.WriteString("; Domain=" + o.Domain)
	}
	if o.Path != "" {
		b.WriteString("; Path=" + o.Path)
	}
	if expires != nil {
		b.WriteString("; Expires=" + expires.UTC().Format(nethttpTimeFormat))
	}
	if o.HttpOnly {
		b.WriteString("; HttpOnly")
	}
	if o.Secure {
		b.WriteString("; Secure")
	}
	if o.Partitioned {
		b.WriteString("; Partitioned")
	}
	switch strings.ToLower(o.Priority) {
	case "low":
		b.WriteString("; Priority=Low")
	case "medium":
		b.WriteString("; Priority=Medium")
	case "high":
		b.WriteString("; Priority=High")
	}
	switch strings.ToLower(o.SameSite) {
	case "lax":
		b.WriteString("; SameSite=Lax")
	case "strict", "true":
		b.WriteString("; SameSite=Strict")
	case "none":
		b.WriteString("; SameSite=None")
	}
	return b.String()
}

const nethttpTimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

// parseCookies mirrors cookie-parser (cookie.parse): first value wins.
func parseCookies(header string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(header, ";") {
		eq := strings.Index(part, "=")
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(part[:eq])
		v := strings.TrimSpace(part[eq+1:])
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = v[1 : len(v)-1]
		}
		if _, ok := out[k]; ok {
			continue
		}
		if dv, err := url.PathUnescape(v); err == nil {
			v = dv
		}
		out[k] = v
	}
	return out
}

// inflateSession mirrors store.createSession + new Session(req, sess): the
// stored values in their order, the cookie revived from its JSON.
func inflateSession(sid string, raw []byte, o *SessionOptions) (*Session, error) {
	v, err := dal.DecodeOrderedJSON(raw)
	if err != nil {
		return nil, err
	}
	rec, ok := v.(*dal.Record)
	if !ok {
		return nil, errors.New("Cannot read properties of undefined (reading 'expires')")
	}
	c, _ := rec.Get("cookie")
	if _, ok := c.(*dal.Record); !ok {
		return nil, errors.New("Cannot read properties of undefined (reading 'expires')")
	}
	sess := &Session{ID: sid, opts: o, Data: dal.NewRecord(), store: o.Store, cookie: parseSessionCookie(c)}
	for _, k := range rec.Keys() {
		if k == "cookie" {
			continue
		}
		x, _ := rec.Get(k)
		sess.Data.Set(k, x)
	}
	return sess, nil
}

// SessionMiddleware mirrors express-session.
func SessionMiddleware(opts SessionOptions) MiddlewareFunction {
	if opts.Store == nil {
		opts.Store = NewMemoryStore()
	}
	if opts.Cookie.Path == "" {
		opts.Cookie.Path = "/"
	}
	o := &opts
	return func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		if req.Session != nil {
			next(nil)
			return nil
		}
		// cookie path must match the request path
		if !strings.HasPrefix(req.URL.Path, o.Cookie.Path) {
			next(nil)
			return nil
		}
		var cookieID string
		if raw, ok := parseCookies(req.Header.Get("Cookie"))[o.Name]; ok && strings.HasPrefix(raw, "s:") {
			if v, ok := unsignCookie(raw[2:], o.Secret); ok {
				cookieID = v
			}
		}

		var sess *Session
		if cookieID != "" {
			raw, ok, err := o.Store.Get(cookieID)
			if err != nil {
				next(err)
				return nil
			}
			if ok {
				if sess, err = inflateSession(cookieID, raw, o); err != nil {
					// JSON.parse errors are reported by the store's get
					next(err)
					return nil
				}
			}
		}
		if sess == nil {
			sess = &Session{ID: uidSafe(), opts: o, Data: dal.NewRecord(), store: o.Store, cookie: newSessionCookie(o.Cookie)}
		}
		req.Session = sess
		originalID := sess.ID
		originalHash := sess.hash()
		touched := false
		touch := func() {
			if !touched {
				sess.cookie.touch()
				touched = true
			}
		}

		isModified := func() bool { return originalID != sess.ID || originalHash != sess.hash() }

		res.OnHeaders(func() {
			if req.Session == nil || sess.destroyed {
				return
			}
			var set bool
			if cookieID != sess.ID {
				set = o.SaveUninitialized || isModified()
			} else {
				set = o.Rolling || (sess.cookie.expires != nil && isModified())
			}
			if !set {
				return
			}
			copts := sess.cookie.serializeOptions()
			if copts.Secure && req.Protocol() != "https" {
				return
			}
			touch()
			res.Header().Add("Set-Cookie", SerializeCookie(o.Name, "s:"+signCookie(sess.ID, o.Secret), copts, sess.cookie.expires, nil))
		})
		res.OnEnd(func() {
			if req.Session == nil || sess.destroyed {
				return
			}
			touch()
			var save bool
			if !o.SaveUninitialized && cookieID != sess.ID {
				save = isModified()
			} else {
				save = o.Resave || isModified()
			}
			if save {
				_ = o.Store.Set(sess.ID, sess.serialize(), sess.Expires())
			} else if cookieID == sess.ID {
				_ = o.Store.Touch(sess.ID, sess.serialize(), sess.Expires())
			}
		})
		next(nil)
		return nil
	}
}

// ClearCookie mirrors res.clearCookie(name, options) of Express 4.
func (r *MedusaResponse) ClearCookie(name string, o SessionCookieOptions) {
	if o.Path == "" {
		o.Path = "/"
	}
	epoch := time.Unix(0, 1e6).UTC()
	r.Header().Add("Set-Cookie", SerializeCookie(name, "", o, &epoch, nil))
}

// CookieOptionsFromConfig converts projectConfig.cookieOptions (as passed to
// res.clearCookie) into cookie options.
func CookieOptionsFromConfig(c *config.CookieOptions) SessionCookieOptions {
	var o SessionCookieOptions
	if c == nil {
		return o
	}
	o.Domain = c.Domain
	o.Path = c.Path
	if c.Secure != nil {
		o.Secure = *c.Secure
	}
	if c.HttpOnly != nil {
		o.HttpOnly = *c.HttpOnly
	}
	if c.Partitioned != nil {
		o.Partitioned = *c.Partitioned
	}
	o.Priority = c.Priority
	switch v := c.SameSite.(type) {
	case string:
		o.SameSite = v
	case bool:
		if v {
			o.SameSite = "strict"
		}
	}
	return o
}
