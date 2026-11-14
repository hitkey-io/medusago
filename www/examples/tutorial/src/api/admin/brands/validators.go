// Package brands holds the /admin/brands API routes.
package brands

import "github.com/hitkey-io/medusago/packages/core/framework/src/zod"

// region: validator
// PostAdminCreateBrand validates the body of POST /admin/brands: an object
// with a `name` string.
var PostAdminCreateBrand = zod.Object(
	zod.F("name", zod.String()),
)

// endregion: validator
