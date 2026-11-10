// Package items mirrors api/admin/order-edits/[id]/items.
package items

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	orderedits "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/order-edits"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/order-edits/:id/items.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, _ := orders.BodyCopy(req).Get("items")
	list, _ := v.([]any)
	items := make([]*dal.Record, 0, len(list))
	for _, x := range list {
		if r, ok := x.(*dal.Record); ok {
			items = append(items, r)
		}
	}
	preview, err := workflows.OrderEditAddNewItemWorkflow.Run(req.Ctx(), req.Scope, workflows.OrderEditItemsInput{OrderID: req.Params["id"], Items: items})
	if err != nil {
		return err
	}
	return orderedits.RespondPreview(res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
