package brands

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// region: middlewares
// Middlewares are the middlewares of the /admin/brands routes.
var Middlewares = []medusahttp.MiddlewareRoute{
	{
		Matcher: "/admin/brands",
		Methods: []string{"POST"},
		Middlewares: []medusahttp.MiddlewareFunction{
			httputils.ValidateAndTransformBody(httputils.Schema(PostAdminCreateBrand)),
		},
	},
}

// endregion: middlewares
