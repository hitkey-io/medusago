// Package id_ mirrors api/admin/orders/[id].
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// GET mirrors GET /admin/orders/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	var version any
	if req.ValidatedQuery != nil {
		version, _ = req.ValidatedQuery.Get("version")
	}
	order, err := workflows.GetOrderDetail(req.Ctx(), req.Scope, workflows.GetOrderDetailInput{
		Fields: req.QueryConfig.Fields, OrderID: req.Params["id"], Version: version,
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	return res.Status(200).JSON(out)
}

// POST mirrors POST /admin/orders/:id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("user_id", orders.ActorID(req))
	in.Set("id", req.Params["id"])
	if _, err := workflows.UpdateOrderWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	f := dal.NewRecord()
	f.Set("id", req.Params["id"])
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "order", Filters: f, Fields: req.QueryConfig.Fields}, joiner.Options{})
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
var Route = medusahttp.Route{GET: GET, POST: POST}
