// Package storecredits mirrors api/store/carts/[id]/store-credits of
// @medusajs/loyalty-plugin.
package storecredits

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	giftcards "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/store/carts/id_/gift-cards"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/carts"
)

// POST mirrors POST /store/carts/:id/store-credits: the store credits of
// the authenticated customer applied to the cart.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	input := carts.AddStoreCreditsToCartWorkflowInput{CartID: id}
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		if amount, ok := body.Get("amount"); ok {
			if f, ok := amount.(float64); ok {
				input.Amount = &f
			}
		}
	}
	if actor := req.AuthString("actor_id"); actor != "" {
		input.CustomerID = &actor
	}
	if _, err := carts.AddStoreCreditsToCartWorkflow.Run(req.Ctx(), req.Scope, input); err != nil {
		return err
	}
	return giftcards.RespondCart(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
