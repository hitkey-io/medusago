package http

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// CorsOrigin is one allowed origin: a string or a regular expression.
type CorsOrigin struct {
	Value string
	Re    *regexp.Regexp
}

var regexpLiteral = regexp.MustCompile(`^([/~@;%#'])(.*?)([/~@;%#'])([gimsuy]*)$`)

// ParseCorsOrigins mirrors parseCorsOrigins / buildRegexpIfValid.
func ParseCorsOrigins(s string) []CorsOrigin {
	if s == "" {
		return nil
	}
	var out []CorsOrigin
	for _, part := range strings.Split(s, ",") {
		if m := regexpLiteral.FindStringSubmatch(part); m != nil && m[1] == m[3] {
			flags := ""
			if strings.Contains(m[4], "i") {
				flags += "i"
			}
			if strings.Contains(m[4], "m") {
				flags += "m"
			}
			if strings.Contains(m[4], "s") {
				flags += "s"
			}
			pattern := m[2]
			if flags != "" {
				pattern = "(?" + flags + ")" + pattern
			}
			if re, err := regexp.Compile(pattern); err == nil {
				out = append(out, CorsOrigin{Re: re})
				continue
			}
		}
		out = append(out, CorsOrigin{Value: part})
	}
	return out
}

func isOriginAllowed(origin string, allowed []CorsOrigin) bool {
	for _, a := range allowed {
		if a.Re != nil {
			if a.Re.MatchString(origin) {
				return true
			}
		} else if a.Value == origin {
			return true
		}
	}
	return false
}

func vary(res *MedusaResponse, field string) {
	cur := res.Header().Get("Vary")
	if cur == "*" {
		return
	}
	if cur == "" {
		res.Header().Set("Vary", field)
		return
	}
	for _, f := range strings.Split(cur, ",") {
		if strings.EqualFold(strings.TrimSpace(f), field) {
			return
		}
	}
	res.Header().Set("Vary", cur+", "+field)
}

// CorsOptions are the cors options taken from projectConfig.http.corsOptions
// (maxAge, exposedHeaders, allowedHeaders, methods). The header lists are a
// string or a list of strings; nil means not set.
type CorsOptions struct {
	MaxAge         *float64
	ExposedHeaders any
	AllowedHeaders any
	Methods        any
}

// joinCorsList mirrors `value.join ? value.join(",") : value`.
func joinCorsList(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return strings.Join(t, ",")
	case []any:
		parts := make([]string, len(t))
		for i, x := range t {
			parts[i] = fmt.Sprint(x)
		}
		return strings.Join(parts, ",")
	}
	return fmt.Sprint(v)
}

// Cors mirrors the cors package with { ...corsOptions, origin: [...],
// credentials: true, preflightContinue: false } as configured by the router.
func Cors(origins []CorsOrigin, opts ...CorsOptions) MiddlewareFunction {
	var o CorsOptions
	if len(opts) > 0 {
		o = opts[0]
	}
	return func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		origin := req.Header.Get("Origin")
		configureOrigin := func() {
			if origin != "" && isOriginAllowed(origin, origins) {
				res.SetHeader("Access-Control-Allow-Origin", origin)
			}
			vary(res, "Origin")
		}
		configureExposedHeaders := func() {
			if o.ExposedHeaders == nil {
				return
			}
			if h := joinCorsList(o.ExposedHeaders); h != "" {
				res.SetHeader("Access-Control-Expose-Headers", h)
			}
		}
		res.SetHeader("Access-Control-Allow-Credentials", "true")
		if req.Method == "OPTIONS" {
			res.Header().Del("Access-Control-Allow-Credentials")
			configureOrigin()
			res.SetHeader("Access-Control-Allow-Credentials", "true")
			methods := "GET,HEAD,PUT,PATCH,POST,DELETE"
			if o.Methods != nil {
				methods = joinCorsList(o.Methods)
			}
			res.SetHeader("Access-Control-Allow-Methods", methods)
			if o.AllowedHeaders == nil {
				if h := req.Header.Get("Access-Control-Request-Headers"); h != "" {
					res.SetHeader("Access-Control-Allow-Headers", h)
				}
				vary(res, "Access-Control-Request-Headers")
			} else if h := joinCorsList(o.AllowedHeaders); h != "" {
				res.SetHeader("Access-Control-Allow-Headers", h)
			}
			if o.MaxAge != nil {
				res.SetHeader("Access-Control-Max-Age", strconv.FormatFloat(*o.MaxAge, 'f', -1, 64))
			}
			configureExposedHeaders()
			res.Status(204)
			res.SetHeader("Content-Length", "0")
			res.End()
			return nil
		}
		res.Header().Del("Access-Control-Allow-Credentials")
		configureOrigin()
		res.SetHeader("Access-Control-Allow-Credentials", "true")
		configureExposedHeaders()
		next(nil)
		return nil
	}
}
