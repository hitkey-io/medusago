// Package imports mirrors api/admin/products/imports.
package imports

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST: the file is normalized, the import waits for the
// confirmation.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	name, _ := body.Get("originalname")
	key, _ := body.Get("file_key")
	result, info, err := workflows.ImportProductsAsChunksWorkflow.RunWithInfo(req.Ctx(), req.Scope, workflows.ImportProductsAsChunksInput{Filename: fmt.Sprint(name), FileKey: fmt.Sprint(key)})
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
