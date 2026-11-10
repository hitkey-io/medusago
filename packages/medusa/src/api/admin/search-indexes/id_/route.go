// Package id mirrors api/admin/search-indexes/[id].
package id

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/search/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/search/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// DELETE mirrors DELETE: deleteSearchIndexWorkflow over the index.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if req.Scope.Resolve(utilsmodulessdk.SEARCH) == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, "The Search Module is not enabled")
	}
	id := req.Params["id"]
	result, err := workflows.DeleteSearchIndexWorkflow.Run(req.Ctx(), req.Scope, steps.SearchIndexesInput{Index: []string{id}})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "search_index")
	out.Set("deleted", true)
	var versions any
	if result != nil {
		versions, _ = result.Get("deleted_versions")
	}
	out.Set("deleted_versions", versions)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{DELETE: DELETE}
