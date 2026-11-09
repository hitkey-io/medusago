package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// This file reproduces express-session 1.x (cookie-signature, uid-safe ids,
// MemoryStore) as configured by the express loader, so that sessions behave
// and serialize exactly like upstream.

// SessionCookieOptions mirrors the express-session cookie options.
type SessionCookieOptions struct {
	Path     string
	Domain   string
	HttpOnly bool
	Secure   bool
	// SameSite: "", "lax", "strict", "none" ("" = false).
	SameSite string
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
// JSON representation of the session (the shape connect-redis stores).
type SessionStore interface {
	Get(sid string) ([]byte, bool, error)
	Set(sid string, data []byte, expires time.Time) error
	Destroy(sid string) error
	Touch(sid string, data []byte, expires time.Time) error
}

// Session mirrors req.session.
type Session struct {
	ID      string
	expires time.Time
	opts    *SessionOptions
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

func (s *Session) cookieJSON() *dal.Record {
	c := dal.NewRecord()
	if s.opts.Cookie.MaxAge > 0 {
		c.Set("originalMaxAge", float64(s.opts.Cookie.MaxAge))
		c.Set("expires", s.expires)
	} else {
		c.Set("originalMaxAge", nil)
		c.Set("expires", nil)
	}
	c.Set("secure", s.opts.Cookie.Secure)
	c.Set("httpOnly", s.opts.Cookie.HttpOnly)
	if s.opts.Cookie.Domain != "" {
		c.Set("domain", s.opts.Cookie.Domain)
	}
	c.Set("path", s.opts.Cookie.Path)
	if s.opts.Cookie.SameSite == "" {
		c.Set("sameSite", false)
	} else {
		c.Set("sameSite", s.opts.Cookie.SameSite)
	}
	return c
}

func (s *Session) serialize() []byte {
	r := dal.NewRecord()
	r.Set("cookie", s.cookieJSON())
	for _, k := range s.Data.Keys() {
		v, _ := s.Data.Get(k)
		r.Set(k, v)
	}
	raw, _ := r.MarshalJSON()
	return raw
}

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

func (m *MemoryStore) Touch(sid string, data []byte, expires time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e, ok := m.sessions[sid]; ok {
		e.expires = expires
		e.data = data
		m.sessions[sid] = e
	}
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

		newSession := func() *Session {
			s := &Session{ID: uidSafe(), opts: o, Data: dal.NewRecord(), store: o.Store}
			if o.Cookie.MaxAge > 0 {
				s.expires = time.Now().Add(time.Duration(o.Cookie.MaxAge) * time.Millisecond)
			}
			return s
		}

		var sess *Session
		if cookieID != "" {
			if raw, ok, err := o.Store.Get(cookieID); err == nil && ok {
				if v, err := dal.DecodeOrderedJSON(raw); err == nil {
					if rec, ok := v.(*dal.Record); ok {
						sess = &Session{ID: cookieID, opts: o, Data: dal.NewRecord(), store: o.Store}
						for _, k := range rec.Keys() {
							if k == "cookie" {
								continue
							}
							x, _ := rec.Get(k)
							sess.Data.Set(k, x)
						}
						// The cookie keeps its original expiry unless rolling.
						sess.expires = time.Now().Add(time.Duration(o.Cookie.MaxAge) * time.Millisecond)
						if c, ok := rec.Get("cookie"); ok {
							if cr, ok := c.(*dal.Record); ok {
								if e, ok := cr.Get("expires"); ok {
									if es, ok := e.(string); ok {
										if t, err := time.Parse(time.RFC3339Nano, es); err == nil {
											sess.expires = t
										}
									}
								}
							}
						}
					}
				}
			}
		}
		if sess == nil {
			sess = newSession()
		}
		req.Session = sess
		originalID := sess.ID
		originalHash := sess.hash()

		isModified := func() bool { return originalID != sess.ID || originalHash != sess.hash() }

		res.OnHeaders(func() {
			if req.Session == nil || sess.destroyed {
				return
			}
			var set bool
			if cookieID != sess.ID {
				set = o.SaveUninitialized || isModified()
			} else {
				set = o.Rolling || (o.Cookie.MaxAge > 0 && isModified())
			}
			if !set {
				return
			}
			if o.Cookie.Secure && req.Protocol() != "https" {
				return
			}
			if o.Rolling && o.Cookie.MaxAge > 0 {
				sess.expires = time.Now().Add(time.Duration(o.Cookie.MaxAge) * time.Millisecond)
			}
			var exp *time.Time
			if o.Cookie.MaxAge > 0 {
				e := sess.expires
				exp = &e
			}
			res.Header().Add("Set-Cookie", SerializeCookie(o.Name, "s:"+signCookie(sess.ID, o.Secret), o.Cookie, exp, nil))
		})
		res.OnEnd(func() {
			if req.Session == nil || sess.destroyed {
				return
			}
			var save bool
			if !o.SaveUninitialized && cookieID != sess.ID {
				save = isModified()
			} else {
				save = o.Resave || isModified()
			}
			if save {
				_ = o.Store.Set(sess.ID, sess.serialize(), sess.expires)
			} else if cookieID == sess.ID {
				_ = o.Store.Touch(sess.ID, sess.serialize(), sess.expires)
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
