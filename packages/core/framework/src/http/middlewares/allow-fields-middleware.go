package middlewares

import medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"

// AllowFields mirrors allowFields(...fields): a middleware adding fields and
// relations to the fields the `fields` query parameter of the routes it is
// applied to may request (validateAndTransformQuery consumes them).
func AllowFields(fields ...[]string) medusahttp.MiddlewareFunction {
	var toAllow []string
	for _, f := range fields {
		toAllow = append(toAllow, f...)
	}
	return func(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		req.Allowed = append(req.Allowed, toAllow...)
		next(nil)
		return nil
	}
}
