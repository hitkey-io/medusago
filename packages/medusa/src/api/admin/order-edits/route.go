package orderedits

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// RespondPreview writes { order_preview }.
func RespondPreview(res *medusahttp.MedusaResponse, preview *dal.Record) error {
	out := dal.NewRecord()
	out.Set("order_preview", preview)
	return res.JSON(out)
}

// POST mirrors POST /admin/order-edits.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	change, err := workflows.BeginOrderEditOrderWorkflow.Run(req.Ctx(), req.Scope, orders.BodyCopy(req))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order_change", change)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
