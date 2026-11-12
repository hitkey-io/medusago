package giftcards

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	gcworkflows "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/workflows/gift-cards/workflows"
)

// GET mirrors GET /admin/gift-cards.
var GET = httputils.RefetchListRoute("gift_cards", "gift_cards")

// RetrieveGiftCard mirrors query.graph({ entity: "gift_cards", fields,
// filters: { id } }, { throwIfKeyNotFound: true }): { gift_card }.
func RetrieveGiftCard(req *medusahttp.MedusaRequest, id any) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", id)
	res, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "gift_cards", IDOrFilter: f, Fields: req.QueryConfig.Fields, Options: joiner.Options{ThrowIfKeyNotFound: true},
	})
	if err != nil {
		return nil, err
	}
	out := dal.NewRecord()
	if len(res.Data) > 0 {
		out.Set("gift_card", res.Data[0])
	} else {
		out.Set("gift_card", nil)
	}
	return out, nil
}

// POST mirrors POST /admin/gift-cards.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	input := dal.NewRecord()
	if body != nil {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			input.Set(k, v)
		}
	}
	giftCards, err := gcworkflows.CreateGiftCardsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{input})
	if err != nil {
		return err
	}
	var id any
	if len(giftCards) > 0 {
		id, _ = giftCards[0].Get("id")
	}
	out, err := RetrieveGiftCard(req, id)
	if err != nil {
		return err
	}
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET, POST: POST}
