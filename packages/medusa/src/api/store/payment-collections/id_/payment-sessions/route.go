// Package paymentsessions mirrors api/store/payment-collections/[id]/payment-sessions.
package paymentsessions

import (
	paymentworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	provider, _ := body.Get("provider_id")
	data, _ := body.Get("data")
	in := paymentworkflows.CreatePaymentSessionsInput{PaymentCollectionID: req.Params["id"]}
	in.ProviderID, _ = provider.(string)
	in.Data, _ = data.(*dal.Record)
	if req.AuthContext != nil {
		actor, _ := req.AuthContext.Get("actor_id")
		in.CustomerID, _ = actor.(string)
	}
	if _, err := paymentworkflows.CreatePaymentSessionsWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
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
