// Package receiveitems mirrors api/admin/returns/[id]/receive-items.
package receiveitems

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/returns"
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

// POST mirrors POST /admin/returns/:id/receive-items.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	b := orders.BodyCopy(req)
	v, _ := b.Get("items")
	preview, err := workflows.ReceiveItemReturnRequestWorkflow.Run(req.Ctx(), req.Scope, workflows.RequestItemReturnInput{ReturnID: req.Params["id"], Items: items(v)})
	if err != nil {
		return err
	}
	return returns.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
