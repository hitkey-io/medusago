// Package export mirrors api/admin/customers/export.
package export

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/customer/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST: the export runs in the background.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	fields := req.QueryConfig.Fields
	if fields == nil {
		fields = []string{}
	}
	in := workflows.ExportCustomersDTO{Select: fields, Filter: req.FilterableFields}
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		if v, ok := body.Get("format"); ok {
			in.Format, _ = v.(string)
		}
		in.BatchSize, _ = body.Get("batch_size")
	}
	if req.AuthContext != nil {
		in.To, _ = req.AuthContext.Get("actor_id")
	}
	_, info, err := workflows.ExportCustomersWorkflow.RunWithInfo(req.Ctx(), req.Scope, in)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("transaction_id", info.TransactionID)
	return res.Status(202).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
