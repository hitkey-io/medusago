// Package converttoorder mirrors api/admin/draft-orders/[id]/convert-to-order.
package converttoorder

import (
	draftworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST /admin/draft-orders/:id/convert-to-order.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := draftworkflows.ConvertDraftOrderWorkflow.Run(req.Ctx(), req.Scope, any(req.Params["id"])); err != nil {
		return err
	}
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "orders", Filters: f, Fields: req.QueryConfig.Fields}, joiner.Options{})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("order", r.Data[0])
	}
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
