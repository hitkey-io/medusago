// Package id_ mirrors api/store/product-options/[id]/route.ts.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	storeutils "github.com/hitkey-io/medusago/packages/medusa/src/api/store/utils"
)

// Route is the route file.
var Route = medusahttp.Route{GET: storeutils.GraphRetrieveRoute("product_option", "product_option", "Product option with id: %s was not found", false)}
