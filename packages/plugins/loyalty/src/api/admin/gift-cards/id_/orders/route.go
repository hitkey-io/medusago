// Package orders mirrors api/admin/gift-cards/[id]/orders of
// @medusajs/loyalty-plugin.
package orders

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET /admin/gift-cards/:id/orders: the orders linked to the
// gift card.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	f := dal.NewRecord()
	f.Set("gift_card_id", req.Params["id"])
	links, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "order_gift_card", IDOrFilter: f, Fields: []string{"id", "gift_card_id", "order_id"},
	})
	if err != nil {
		return err
	}
	orderIDs := make([]any, 0, len(links.Data))
	for _, l := range links.Data {
		if r, ok := l.(*dal.Record); ok {
			id, _ := r.Get("order_id")
			orderIDs = append(orderIDs, id)
		}
	}
	of := dal.NewRecord()
	of.Set("id", orderIDs)
	orders, err := httputils.RefetchEntities(req.Ctx(), req.Scope, httputils.RefetchEntitiesInput{
		Entity: "order", IDOrFilter: of, Fields: req.QueryConfig.Fields,
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	data := orders.Data
	if data == nil {
		data = []any{}
	}
	out.Set("orders", data)
	return res.JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{GET: GET}
