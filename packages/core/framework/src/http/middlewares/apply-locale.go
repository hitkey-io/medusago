package middlewares

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// NormalizeLocale mirrors normalizeLocale.
func NormalizeLocale(locale string) string { return common.NormalizeLocale(locale) }

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
