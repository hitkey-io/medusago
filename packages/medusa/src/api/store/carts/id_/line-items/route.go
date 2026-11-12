// Package lineitems mirrors api/store/carts/[id]/line-items.
package lineitems

import (
	cartworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/cart/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/store/carts"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	var additional *dal.Record
	if body != nil {
		additional, _ = carts.Get(body, "additional_data").(*dal.Record)
	}
	if _, err := cartworkflows.AddToCartWorkflow.Run(req.Ctx(), req.Scope, cartworkflows.AddToCartInput{CartID: req.Params["id"], Items: []*dal.Record{body}, AdditionalData: additional}, composer.RunOptions{Notify: true}); err != nil {
		return err
	}
	return carts.RespondCart(req, res, req.Params["id"])
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
