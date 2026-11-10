// Package id_ mirrors api/admin/orders/[id].
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/orders/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	var version any
	if req.ValidatedQuery != nil {
		version, _ = req.ValidatedQuery.Get("version")
	}
	order, err := workflows.GetOrderDetail(req.Ctx(), httputils.QueryOf(req.Scope), workflows.GetOrderDetailInput{
		Fields: req.QueryConfig.Fields, OrderID: req.Params["id"], Version: version,
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
