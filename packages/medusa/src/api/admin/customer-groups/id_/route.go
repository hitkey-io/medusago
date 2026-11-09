// Package id_ mirrors api/admin/customer-groups/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
)

// Route is the route file.
var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("customer_group", "customer_group", "Customer group with id: %s not found")}
