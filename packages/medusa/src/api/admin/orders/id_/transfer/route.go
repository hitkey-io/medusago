// Package transfer mirrors api/admin/orders/[id]/transfer.
package transfer

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/transfer.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	b := orders.BodyCopy(req)
	in := dal.NewRecord()
	in.Set("order_id", req.Params["id"])
	for _, k := range []string{"customer_id"} {
		v, _ := b.Get(k)
		in.Set(k, v)
	}
	in.Set("logged_in_user", orders.ActorID(req))
	for _, k := range []string{"description", "internal_note", "update_order_email"} {
		if v, ok := b.Get(k); ok {
			in.Set(k, v)
		}
	}
	if _, err := workflows.RequestOrderTransferWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return orders.RespondOrder(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
