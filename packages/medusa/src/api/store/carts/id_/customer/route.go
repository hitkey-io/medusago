// Package customer mirrors api/store/carts/[id]/customer.
package customer

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

// POST mirrors POST /store/carts/:id/customer.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := cartworkflows.TransferCartCustomerInput{ID: req.Params["id"]}
	if req.AuthContext != nil {
		in.CustomerID, _ = req.AuthContext.Get("actor_id")
	}
	if b, ok := req.ValidatedBody.(*dal.Record); ok && b != nil {
		v, _ := b.Get("additional_data")
		in.AdditionalData, _ = v.(*dal.Record)
	}
	if _, err := cartworkflows.TransferCartCustomerWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return carts.RespondCart(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
