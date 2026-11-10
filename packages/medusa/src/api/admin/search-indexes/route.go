// Package searchindexes mirrors api/admin/search-indexes.
package searchindexes

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type indexLister interface {
	ListIndexes(ctx context.Context) ([]*dal.Record, error)
}

// GET mirrors GET: the registered search indexes (the search module is
// registered by default; the default app defines no search index).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	out := dal.NewRecord()
	searchModule, _ := req.Scope.Resolve(utilsmodulessdk.SEARCH).(indexLister)
	if searchModule == nil {
		out.Set("search_indexes", []any{})
		out.Set("enabled", false)
		return res.JSON(out)
	}
	indexes, err := searchModule.ListIndexes(req.Ctx())
	if err != nil {
		return err
	}
	out.Set("search_indexes", indexes)
	out.Set("enabled", true)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
