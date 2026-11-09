package http

import (
	"regexp"
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

// Cors mirrors the cors package with { origin: [...], credentials: true,
// preflightContinue: false } as configured by the router.
func Cors(origins []CorsOrigin) MiddlewareFunction {
	return func(req *MedusaRequest, res *MedusaResponse, next NextFunction) error {
		origin := req.Header.Get("Origin")
		configureOrigin := func() {
			if origin != "" && isOriginAllowed(origin, origins) {
				res.SetHeader("Access-Control-Allow-Origin", origin)
			}
			vary(res, "Origin")
		}
		res.SetHeader("Access-Control-Allow-Credentials", "true")
		if req.Method == "OPTIONS" {
			res.Header().Del("Access-Control-Allow-Credentials")
			configureOrigin()
			res.SetHeader("Access-Control-Allow-Credentials", "true")
			res.SetHeader("Access-Control-Allow-Methods", "GET,HEAD,PUT,PATCH,POST,DELETE")
			if h := req.Header.Get("Access-Control-Request-Headers"); h != "" {
				res.SetHeader("Access-Control-Allow-Headers", h)
			}
			vary(res, "Access-Control-Request-Headers")
			res.Status(204)
			res.SetHeader("Content-Length", "0")
			res.End()
			return nil
		}
		res.Header().Del("Access-Control-Allow-Credentials")
		configureOrigin()
		res.SetHeader("Access-Control-Allow-Credentials", "true")
		next(nil)
		return nil
	}
}
