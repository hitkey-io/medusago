// Package giftcards mirrors api/store/carts/[id]/gift-cards of
// @medusajs/loyalty-plugin.
package giftcards

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/carts"
)

func code(v any) string {
	r, _ := v.(*dal.Record)
	if r == nil {
		return ""
	}
	c, _ := r.Get("code")
	s, _ := c.(string)
	return s
}

// RespondCart mirrors query.graph({ entity: "cart", fields, filters: { id }
// }, { throwIfKeyNotFound: true }) answered as { cart }.
func RespondCart(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, id string) error {
	f := dal.NewRecord()
	f.Set("id", id)
	r, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "cart", IDOrFilter: f, Fields: req.QueryConfig.Fields, Options: joiner.Options{ThrowIfKeyNotFound: true},
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if len(r.Data) > 0 {
		out.Set("cart", r.Data[0])
	} else {
		out.Set("cart", nil)
	}
	return res.JSON(out)
}

// POST mirrors POST /store/carts/:id/gift-cards.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := carts.AddGiftCardToCartWorkflow.Run(req.Ctx(), req.Scope, carts.AddGiftCardToCartWorkflowInput{CartID: id, Code: code(req.ValidatedBody)}); err != nil {
		return err
	}
	return RespondCart(req, res, id)
}

// DELETE mirrors DELETE /store/carts/:id/gift-cards (the workflow input is
// the raw body).
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := carts.RemoveGiftCardFromCartWorkflow.Run(req.Ctx(), req.Scope, carts.RemoveGiftCardFromCartWorkflowInput{CartID: id, Code: code(req.Body)}); err != nil {
		return err
	}
	return RespondCart(req, res, id)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
