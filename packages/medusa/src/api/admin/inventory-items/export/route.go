// Package export mirrors api/admin/inventory-items/export.
package export

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST: the export runs in the background.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	fields := req.QueryConfig.Fields
	if fields == nil {
		fields = []string{}
	}
	_, info, err := workflows.ExportInventoryItemsWorkflow.RunWithInfo(req.Ctx(), req.Scope, steps.ExportInventoryItemsDTO{Select: fields, Filter: req.FilterableFields})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("transaction_id", info.TransactionID)
	return res.Status(202).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
