// Package markaspaid mirrors api/admin/payment-collections/[id]/mark-as-paid.
package markaspaid

import (
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	paymentcollections "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/payment-collections"
)

// POST mirrors POST /admin/payment-collections/:id/mark-as-paid.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("payment_collection_id", req.Params["id"])
	in.Set("captured_by", orders.ActorID(req))
	if _, err := orderworkflows.MarkPaymentCollectionAsPaid.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return paymentcollections.Respond(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
