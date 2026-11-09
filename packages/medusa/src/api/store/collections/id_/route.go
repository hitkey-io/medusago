// Package id_ mirrors api/store/collections/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
)

// Route is the route file.
var Route = medusahttp.Route{GET: storeutils.GraphRetrieveRoute("product_collection", "collection", "Collection with id: %s was not found", true)}
