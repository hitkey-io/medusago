// Package products holds the middlewares the project adds to Medusa's
// /admin/products API routes.
package products

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// region: validator
// additionalData describes the custom properties accepted in the
// `additional_data` of POST /admin/products.
var additionalData = zod.Object(
	zod.F("brand_id", zod.String().Optional()),
)

// ValidateAdditionalData validates the `additional_data` of the request
// body. It runs after the validation middleware of the core route, so
// the body is already validated.
func ValidateAdditionalData(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	if body == nil {
		next(nil)
		return nil
	}
	if data, _ := body.Get("additional_data"); data != nil {
		if _, err := zod.ZodValidator(additionalData, data); err != nil {
			return err
		}
	}
	next(nil)
	return nil
}

// endregion: validator

// region: middlewares
// Middlewares are the middlewares added to the /admin/products routes.
var Middlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/admin/products",
		Methods: []string{"POST"},
		Middlewares: []medusahttp.MiddlewareFunction{
			ValidateAdditionalData,
		},
	},
}

// endregion: middlewares
