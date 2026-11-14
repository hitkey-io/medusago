// Package api registers the API routes and middlewares of the project.
package api

import (
	"slices"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/api/admin/brands"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/api/admin/products"
)

// region: middlewares
// Middlewares are the middlewares of the project.
var Middlewares = slices.Concat[[]medusahttp.MiddlewareRoute](
	brands.Middlewares,
	products.Middlewares,
)

// endregion: middlewares
