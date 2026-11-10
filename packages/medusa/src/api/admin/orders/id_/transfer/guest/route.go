// Package guest mirrors api/admin/orders/[id]/transfer/guest.
package guest

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/orders/:id/transfer/guest.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	b := orders.BodyCopy(req)
	in := dal.NewRecord()
	in.Set("order_id", req.Params["id"])
	v, _ := b.Get("email")
	in.Set("email", v)
	in.Set("logged_in_user", orders.ActorID(req))
	for _, k := range []string{"description", "internal_note"} {
		if v, ok := b.Get(k); ok {
			in.Set(k, v)
		}
	}
	if _, err := workflows.TransferOrderToGuestWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return orders.RespondOrder(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
