// Package cancel mirrors api/admin/exchanges/[id]/cancel.
package cancel

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/exchanges/:id/cancel: the workflow returns nothing, so
// the body is { exchange: undefined } = {}.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("exchange_id", req.Params["id"])
	in.Set("canceled_by", orders.ActorID(req))
	if _, err := workflows.CancelOrderExchangeWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return res.Status(200).JSON(dal.NewRecord())
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
