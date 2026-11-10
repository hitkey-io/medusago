// Package paymentsessions mirrors api/admin/payment-collections/[id]/payment-sessions.
package paymentsessions

import (
	pcworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/payment-collection/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	paymentcollections "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payment-collections"
)

// POST mirrors POST /admin/payment-collections/:id/payment-sessions.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body := orders.BodyCopy(req)
	provider, _ := body.Get("provider_id")
	data, _ := body.Get("data")
	d, _ := data.(*dal.Record)
	p, _ := provider.(string)
	if _, err := pcworkflows.CreatePaymentSessionsWorkflow.Run(req.Ctx(), req.Scope, pcworkflows.CreatePaymentSessionsInput{PaymentCollectionID: req.Params["id"], ProviderID: p, Data: d}); err != nil {
		return err
	}
	return paymentcollections.Respond(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
