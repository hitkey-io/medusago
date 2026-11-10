// Package claimitems mirrors api/admin/claims/[id]/claim-items.
package claimitems

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/claims"
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

// POST mirrors POST /admin/claims/:id/claim-items.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	v, _ := orders.BodyCopy(req).Get("items")
	preview, err := workflows.OrderClaimItemWorkflow.Run(req.Ctx(), req.Scope, workflows.OrderClaimItemsInput{ClaimID: req.Params["id"], Items: items(v)})
	if err != nil {
		return err
	}
	return claims.RespondPreview(req, res, preview)
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
