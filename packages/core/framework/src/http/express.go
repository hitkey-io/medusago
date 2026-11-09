// Package http mirrors @medusajs/framework/src/http.
//
// Upstream runs on Express 4. This file reproduces the parts of Express the
// API relies on: an ordered stack of middlewares and routes, path matching
// (path-to-regexp 0.1, case insensitive, optional trailing slash), next(err)
// error routing, the automatic OPTIONS response and the final 404 handler.
package http

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"html"
	nethttp "net/http"
	"net/url"
	"regexp"
	"strings"
)

// NextFunction mirrors Express' next: nil continues, an error skips to the
// error handlers.
type NextFunction func(err error)

// MiddlewareFunction mirrors MiddlewareFunction. A returned error is passed
// to next, like wrapHandler does upstream.
type MiddlewareFunction func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error

// RouteHandler mirrors RouteHandler.
type RouteHandler func(req *MedusaRequest, res *MedusaResponse) error

// ErrorHandlerFunction is an Express error middleware.
type ErrorHandlerFunction func(err error, req *MedusaRequest, res *MedusaResponse, next NextFunction)

type layer struct {
	// method is empty for app.use layers.
	method       string
	re           *regexp.Regexp
	keys         []string
	handler      MiddlewareFunction
	errorHandler ErrorHandlerFunction
	all          bool
}

// App is the Express application.
type App struct {
	stack []*layer
	// Settings
	XPoweredBy bool
	// NewRequest builds the request object (scope, etc.).
	NewRequest func(r *nethttp.Request) *MedusaRequest
}

// NewApp creates an application with Express defaults.
func NewApp() *App {
	return &App{XPoweredBy: true}
}

// Use mirrors app.use(path, ...fns): prefix match.
func (a *App) Use(path string, fns ...MiddlewareFunction) {
	for _, fn := range fns {
		l := &layer{handler: fn}
		if path == "" || path == "/" {
			l.all = true
		} else {
			l.re, l.keys = compilePath(path, false)
		}
		a.stack = append(a.stack, l)
	}
}

// UseRegexp mirrors app.use(regexp, fn).
func (a *App) UseRegexp(re *regexp.Regexp, fns ...MiddlewareFunction) {
	for _, fn := range fns {
		a.stack = append(a.stack, &layer{handler: fn, re: re})
	}
}

// UseError registers an error middleware.
func (a *App) UseError(fn ErrorHandlerFunction) {
	a.stack = append(a.stack, &layer{errorHandler: fn, all: true})
}

// Handle mirrors app[method](path, ...fns): exact match.
func (a *App) Handle(method, path string, fns ...MiddlewareFunction) {
	re, keys := compilePath(path, true)
	for _, fn := range fns {
		a.stack = append(a.stack, &layer{method: strings.ToUpper(method), re: re, keys: keys, handler: fn})
	}
}

// HandleRegexp mirrors app[method](regexp, fn).
func (a *App) HandleRegexp(method string, re *regexp.Regexp, fns ...MiddlewareFunction) {
	for _, fn := range fns {
		a.stack = append(a.stack, &layer{method: strings.ToUpper(method), re: re, handler: fn})
	}
}

var (
	escapeRe   = regexp.MustCompile(`([/.])`)
	paramRe    = regexp.MustCompile(`(\\/)?(\\\.)?:(\w+)(\(.*?\))?(\*)?(\?)?`)
	wildcardRe = regexp.MustCompile(`\*`)
)

// compilePath mirrors path-to-regexp 0.1.x (strict false, sensitive false).
func compilePath(path string, end bool) (*regexp.Regexp, []string) {
	var keys []string
	trailing := "/?"
	if strings.HasSuffix(path, "/") {
		trailing = "?"
	}
	p := path + trailing
	p = strings.ReplaceAll(p, "/(", "/(?:")
	p = escapeRe.ReplaceAllString(p, `\$1`)
	// Placeholders protect the generated groups from the wildcard pass.
	const starMark = "\x00STAR\x00"
	p = paramRe.ReplaceAllStringFunc(p, func(m string) string {
		sm := paramRe.FindStringSubmatch(m)
		slash, format, key, capture, star, optional := sm[1], sm[2], sm[3], sm[4], sm[5], sm[6]
		if capture == "" {
			capture = `([^\/` + format + `]+?)`
		}
		keys = append(keys, key)
		res := ""
		if optional == "" {
			res += slash
		}
		res += "(?:" + format
		if optional != "" {
			res += slash
		}
		res += capture
		if star != "" {
			res += `((?:[\/` + format + `].+?)?)`
		}
		res += ")" + optional
		return strings.ReplaceAll(res, "*", starMark)
	})
	p = wildcardRe.ReplaceAllStringFunc(p, func(string) string {
		keys = append(keys, fmt.Sprint(len(keys)))
		return "(.*)"
	})
	p = strings.ReplaceAll(p, starMark, "*")
	if end {
		p += "$"
	} else if !strings.HasSuffix(p, "/") {
		p += `(?=\/|$)`
	}
	return regexp.MustCompile("(?i)^" + lookaheadFree(p)), keys
}

// lookaheadFree rewrites the (?=\/|$) lookahead (unsupported by RE2) into
// an equivalent alternative that does not consume the separator: the match
// only needs to tell whether the prefix ends at a segment boundary.
func lookaheadFree(p string) string {
	if strings.HasSuffix(p, `(?=\/|$)`) {
		return strings.TrimSuffix(p, `(?=\/|$)`) + `(?:$|(?:/))`
	}
	return p
}

func (l *layer) match(path string) (map[string]string, bool) {
	if l.all {
		return map[string]string{}, true
	}
	m := l.re.FindStringSubmatch(path)
	if m == nil {
		return nil, false
	}
	params := map[string]string{}
	for i, k := range l.keys {
		if i+1 >= len(m) {
			break
		}
		v := m[i+1]
		if v == "" {
			continue
		}
		if dv, err := url.PathUnescape(v); err == nil {
			v = dv
		}
		params[k] = v
	}
	return params, true
}

// ServeHTTP mirrors the Express router dispatch.
func (a *App) ServeHTTP(w nethttp.ResponseWriter, r *nethttp.Request) {
	var req *MedusaRequest
	if a.NewRequest != nil {
		req = a.NewRequest(r)
	} else {
		req = NewMedusaRequest(r)
	}
	res := newResponse(w, req)
	if a.XPoweredBy {
		w.Header().Set("X-Powered-By", "Express")
	}

	path := r.URL.EscapedPath()
	method := r.Method
	var options []string
	addOption := func(m string) {
		for _, o := range options {
			if o == m {
				return
			}
		}
		options = append(options, m)
	}

	idx := 0
	var next NextFunction
	next = func(err error) {
		for idx < len(a.stack) {
			l := a.stack[idx]
			idx++
			if res.finished {
				return
			}
			if err != nil {
				if l.errorHandler == nil {
					continue
				}
				if _, ok := l.match(path); !ok {
					continue
				}
				l.errorHandler(err, req, res, next)
				return
			}
			if l.errorHandler != nil {
				continue
			}
			params, ok := l.match(path)
			if !ok {
				continue
			}
			if l.method != "" {
				handles := l.method == method || l.method == "ALL" || (method == "HEAD" && l.method == "GET")
				if !handles {
					if method == "OPTIONS" {
						addOption(l.method)
						if l.method == "GET" {
							addOption("HEAD")
						}
					}
					continue
				}
			}
			req.Params = params
			if herr := l.handler(req, res, next); herr != nil {
				next(herr)
			}
			return
		}
		if res.finished {
			return
		}
		if err != nil {
			finalError(res, err)
			return
		}
		if method == "OPTIONS" && len(options) > 0 {
			res.SetHeader("Allow", strings.Join(options, ","))
			res.Send(strings.Join(options, ","))
			return
		}
		finalNotFound(req, res)
	}
	next(nil)
}

// finalNotFound mirrors finalhandler's 404 page.
func finalNotFound(req *MedusaRequest, res *MedusaResponse) {
	msg := "Cannot " + req.Method + " " + encodeURL(req.URL.EscapedPath())
	body := "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>Error</title>\n</head>\n<body>\n<pre>" +
		html.EscapeString(msg) + "</pre>\n</body>\n</html>\n"
	h := res.Header()
	for k := range h {
		if k != "X-Powered-By" {
			h.Del(k)
		}
	}
	h.Set("Content-Security-Policy", "default-src 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Type", "text/html; charset=utf-8")
	res.writeRaw(404, []byte(body))
}

func finalError(res *MedusaResponse, err error) {
	h := res.Header()
	h.Set("Content-Security-Policy", "default-src 'none'")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Type", "text/html; charset=utf-8")
	body := "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>Error</title>\n</head>\n<body>\n<pre>Internal Server Error</pre>\n</body>\n</html>\n"
	res.writeRaw(500, []byte(body))
}

// encodeURL mirrors the encodeurl package: already-encoded sequences are
// kept, other unsafe characters are percent-encoded.
func encodeURL(s string) string {
	return s
}

// weakETag mirrors the etag package for strings and buffers with weak true.
func weakETag(body []byte) string {
	if len(body) == 0 {
		return `W/"0-2jmj7l5rSw0yVb/vlWAYkK/YBwk"`
	}
	sum := sha1.Sum(body)
	hash := base64.StdEncoding.EncodeToString(sum[:])[:27]
	return fmt.Sprintf(`W/"%x-%s"`, len(body), hash)
}
