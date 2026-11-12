// Package id_ mirrors api/admin/gift-cards/[id] of @medusajs/loyalty-plugin.
package id_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	giftcards "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/api/admin/gift-cards"
	gcworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/workflows"
)

// GET mirrors GET /admin/gift-cards/:id.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	out, err := giftcards.RetrieveGiftCard(req, req.Params["id"])
	if err != nil {
		return err
	}
	return res.JSON(out)
}

// POST mirrors POST /admin/gift-cards/:id: the update is the raw body
// (req.body) with the id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := giftcards.RetrieveGiftCard(req, id); err != nil {
		return err
	}
	update := dal.NewRecord()
	update.Set("id", id)
	if body, ok := req.Body.(*dal.Record); ok {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			update.Set(k, v)
		}
	}
	if _, err := gcworkflows.UpdateGiftCardsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{update}); err != nil {
		return err
	}
	out, err := giftcards.RetrieveGiftCard(req, id)
	if err != nil {
		return err
	}
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
