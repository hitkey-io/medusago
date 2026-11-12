// Package decline mirrors api/store/orders/[id]/transfer/decline.
package decline

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

func body(req *medusahttp.MedusaRequest, key string) (any, bool) {
	b, _ := req.ValidatedBody.(*dal.Record)
	if b == nil {
		return nil, false
	}
	return b.Get(key)
}

func actor(req *medusahttp.MedusaRequest, key string) any {
	if req.AuthContext == nil {
		return nil
	}
	v, _ := req.AuthContext.Get(key)
	return v
}

// POST mirrors POST /store/orders/:id/transfer/decline.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := dal.NewRecord()
	in.Set("order_id", req.Params["id"])
	v, _ := body(req, "token")
	in.Set("token", v)
	if _, err := workflows.DeclineOrderTransferRequestWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	order, err := workflows.GetOrderDetail(req.Ctx(), req.Scope, workflows.GetOrderDetailInput{
		Fields: req.QueryConfig.Fields, OrderID: req.Params["id"],
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("order", order)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
