// Package id_ mirrors api/admin/users/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// Route is the route file (GET; the remote query variables { id } act as
// the id filter).
var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("user", "user", "User with id: %s was not found")}
