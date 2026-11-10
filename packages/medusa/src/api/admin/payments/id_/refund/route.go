// Package refund mirrors api/admin/payments/[id]/refund.
package refund

import (
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments"
)

// POST mirrors POST /admin/payments/:id/refund.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := dal.NewRecord()
	in.Set("payment_id", req.Params["id"])
	in.Set("created_by", orders.ActorID(req))
	body := orders.BodyCopy(req)
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	if _, err := paymentworkflows.RefundPaymentWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return payments.Respond(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
