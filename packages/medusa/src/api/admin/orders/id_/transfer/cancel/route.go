// Package cancel mirrors api/admin/orders/[id]/transfer/cancel.
package cancel

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/transfer/cancel.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := dal.NewRecord()
	in.Set("order_id", req.Params["id"])
	in.Set("logged_in_user_id", orders.ActorID(req))
	var actorType any
	if req.AuthContext != nil {
		actorType, _ = req.AuthContext.Get("actor_type")
	}
	in.Set("actor_type", actorType)
	if _, err := workflows.CancelOrderTransferRequestWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
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
var Route = medusahttp.Route{POST: POST}
