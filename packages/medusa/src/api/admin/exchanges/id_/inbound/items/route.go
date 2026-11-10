// Package items mirrors api/admin/exchanges/[id]/inbound/items.
package items

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/exchanges"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

func items(v any) []*dal.Record {
	list, _ := v.([]any)
	out := make([]*dal.Record, 0, len(list))
	for _, x := range list {
		if r, ok := x.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

// POST mirrors POST /admin/exchanges/:id/inbound/items.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := exchanges.ExchangeReturnID(req); err != nil {
		return err
	}
	b := orders.BodyCopy(req)
	v, _ := b.Get("items")
	loc, _ := b.Get("location_id")
	preview, err := workflows.OrderExchangeRequestItemReturnWorkflow.Run(req.Ctx(), req.Scope, workflows.OrderExchangeItemsInput{ExchangeID: req.Params["id"], Items: items(v), LocationID: loc})
	if err != nil {
		return err
	}
	return exchanges.RespondReturnPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
