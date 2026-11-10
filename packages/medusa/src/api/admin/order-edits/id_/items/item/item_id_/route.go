// Package item_id_ mirrors api/admin/order-edits/[id]/items/item/[item_id].
package item_id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	orderedits "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/order-edits/:id/items/item/:item_id.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	item := orders.BodyCopy(req)
	item.Set("id", req.Params["item_id"])
	preview, err := workflows.OrderEditUpdateItemQuantityWorkflow.Run(req.Ctx(), req.Scope, workflows.OrderEditItemsInput{OrderID: req.Params["id"], Items: []*dal.Record{item}})
	if err != nil {
		return err
	}
	return orderedits.RespondPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
