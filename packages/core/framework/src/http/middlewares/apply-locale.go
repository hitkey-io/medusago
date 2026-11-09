package middlewares

import (
	"regexp"
	"strings"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
)

var scriptSubtag = regexp.MustCompile(`^[a-zA-Z]{4}$`)

func upperCaseFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// NormalizeLocale mirrors normalizeLocale.
func NormalizeLocale(locale string) string {
	segments := strings.Split(locale, "-")
	switch len(segments) {
	case 1:
		return strings.ToLower(segments[0])
	case 2:
		if scriptSubtag.MatchString(segments[1]) {
			return strings.ToLower(segments[0]) + "-" + upperCaseFirst(strings.ToLower(segments[1]))
		}
		return strings.ToLower(segments[0]) + "-" + strings.ToUpper(segments[1])
	case 3:
		return strings.ToLower(segments[0]) + "-" + upperCaseFirst(strings.ToLower(segments[1])) + "-" + strings.ToUpper(segments[2])
	}
	return locale
}

// ApplyLocale mirrors applyLocale: ?locale= (removed from the query) or the
// x-medusa-locale header.
func ApplyLocale(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	if req.Query != nil {
		if v, ok := req.Query.Get("locale"); ok {
			if s, ok := v.(string); ok && s != "" {
				req.Locale = NormalizeLocale(s)
				req.Query.Delete("locale")
				next(nil)
				return nil
			}
		}
	}
	if h := req.Header.Get("x-medusa-locale"); h != "" {
		req.Locale = NormalizeLocale(h)
	}
	next(nil)
	return nil
}
