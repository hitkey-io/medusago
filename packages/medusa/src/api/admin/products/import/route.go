// Package importroute mirrors api/admin/products/import (the legacy import).
package importroute

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if req.File == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "No file was uploaded for importing")
	}
	result, info, err := workflows.ImportProductsWorkflow.RunWithInfo(req.Ctx(), req.Scope, workflows.ImportProductsInput{Filename: req.File.OriginalName, FileContent: string(req.File.Buffer)})
	if err != nil {
		return err
	}
	summary := dal.NewRecord()
	summary.Set("toCreate", float64(result.ToCreate))
	summary.Set("toUpdate", float64(result.ToUpdate))
	out := dal.NewRecord()
	out.Set("transaction_id", info.TransactionID)
	out.Set("summary", summary)
	return res.Status(202).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
