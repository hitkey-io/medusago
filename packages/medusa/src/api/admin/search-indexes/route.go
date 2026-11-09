// Package searchindexes mirrors api/admin/search-indexes.
package searchindexes

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET. The search module is registered by default
// (defineConfig) and the default app defines no search index, so
// listIndexes returns [] (index definitions are not ported yet).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	out := dal.NewRecord()
	out.Set("search_indexes", []any{})
	out.Set("enabled", true)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
