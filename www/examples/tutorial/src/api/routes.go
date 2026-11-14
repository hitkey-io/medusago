package api

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/api/admin/brands"
)

// region: routes
// Routes are the API routes of the project.
var Routes = []medusahttp.RouteFile{
	{Path: "/admin/brands", Route: brands.Route},
}

// endregion: routes
