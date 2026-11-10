// Package itemid mirrors api/admin/draft-orders/[id]/edit/items/item/[item_id].
package itemid

import (
	dw "github.com/hitkey-io/medusago/packages/core/core-flows/src/draft-order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	draftorders "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/draft-orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/draft-orders/:id/edit/items/item/:item_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	item := orders.BodyCopy(req)
	item.Set("id", req.Params["item_id"])
	preview, err := dw.UpdateDraftOrderItemWorkflow.Run(req.Ctx(), req.Scope, dw.DraftOrderItemsInput{OrderID: req.Params["id"], Items: []*dal.Record{item}})
	if err != nil {
		return err
	}
	return draftorders.RespondDraftPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
