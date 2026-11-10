// Package sessionid mirrors api/store/payment-collections/[id]/payment-sessions/[session_id].
package sessionid

import (
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST /store/payment-collections/:id/payment-sessions/:session_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := paymentworkflows.UpdatePaymentSessionClientDataInput{PaymentCollectionID: req.Params["id"], PaymentSessionID: req.Params["session_id"]}
	if b, ok := req.ValidatedBody.(*dal.Record); ok && b != nil {
		in.ClientData, _ = b.Get("client_data")
	}
	if _, err := paymentworkflows.UpdatePaymentSessionClientDataWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	pc, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "payment_collection", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("payment_collection", pc)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
