// Package export mirrors api/admin/products/export.
package export

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/product/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/products"
)

// POST mirrors POST: the export runs in the background.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	fields := req.QueryConfig.Fields
	if fields == nil {
		fields = []string{}
	}
	_, info, err := workflows.ExportProductsWorkflow.RunWithInfo(req.Ctx(), req.Scope, steps.ExportProductsDTO{Select: products.RemapKeysForProduct(fields), Filter: req.FilterableFields})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("transaction_id", info.TransactionID)
	return res.Status(202).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
