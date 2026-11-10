// Package reindex mirrors api/admin/search-indexes/[id]/reindex.
package reindex

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/search/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/search/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// POST mirrors POST: reindexSearchIndexesWorkflow (the reindex runs in the
// background), answered 202 with the transaction id as job id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if req.Scope.Resolve(utilsmodulessdk.SEARCH) == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "The Search Module is not enabled")
	}
	id := req.Params["id"]
	in := steps.SearchIndexesInput{Index: []string{id}}
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		if v, ok := body.Get("since"); ok && v != nil {
			s := fmt.Sprint(v)
			in.Since = &s
		}
		if v, ok := body.Get("filters"); ok {
			in.Filters, _ = v.(*dal.Record)
		}
		if v, ok := body.Get("strategy"); ok && v != nil {
			s := fmt.Sprint(v)
			in.Strategy = &s
		}
	}
	_, info, err := workflows.ReindexSearchIndexesWorkflow.RunWithInfo(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("job_id", info.TransactionID)
	out.Set("indexes", []any{id})
	return res.Status(202).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
