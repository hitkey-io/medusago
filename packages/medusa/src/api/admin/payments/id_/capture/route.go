// Package capture mirrors api/admin/payments/[id]/capture.
package capture

import (
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payments"
)

// POST mirrors POST /admin/payments/:id/capture.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body := orders.BodyCopy(req)
	in := dal.NewRecord()
	in.Set("payment_id", req.Params["id"])
	in.Set("captured_by", orders.ActorID(req))
	if v, ok := body.Get("amount"); ok {
		in.Set("amount", v)
	}
	if _, err := paymentworkflows.CapturePaymentWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return payments.Respond(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
