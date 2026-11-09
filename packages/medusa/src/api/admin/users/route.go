package users

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// Route is route.ts.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("user", "users")}
