// Package dismissitems mirrors api/admin/returns/[id]/dismiss-items.
package dismissitems

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
)

// POST mirrors POST /admin/returns/:id/dismiss-items.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, _ := orders.BodyCopy(req).Get("items")
	list, _ := v.([]any)
	items := make([]*dal.Record, 0, len(list))
	for _, x := range list {
		if r, ok := x.(*dal.Record); ok {
			items = append(items, r)
		}
	}
	preview, err := workflows.DismissItemReturnRequestWorkflow.Run(req.Ctx(), req.Scope, workflows.RequestItemReturnInput{ReturnID: req.Params["id"], Items: items})
	if err != nil {
		return err
	}
	return returns.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
