// Package id_ mirrors api/store/regions/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// Route is the route file.
var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("region", "region", "Region with id: %s was not found")}
