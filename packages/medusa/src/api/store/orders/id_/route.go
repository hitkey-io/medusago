// Package id_ mirrors api/store/orders/[id].
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /store/orders/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	filters := dal.NewRecord()
	filters.Set("is_draft_order", false)
	order, err := workflows.GetOrderDetail(req.Ctx(), req.Scope, workflows.GetOrderDetailInput{
		Fields: req.QueryConfig.Fields, OrderID: req.Params["id"], Filters: filters,
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
