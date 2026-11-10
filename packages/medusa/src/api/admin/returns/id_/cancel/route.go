// Package cancel mirrors api/admin/returns/[id]/cancel.
package cancel

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/orders"
)

// POST mirrors POST /admin/returns/:id/cancel: the workflow returns nothing,
// so the body is { return: undefined } = {}.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	in := orders.BodyCopy(req)
	in.Set("return_id", req.Params["id"])
	if _, err := workflows.CancelReturnWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	return res.Status(200).JSON(dal.NewRecord())
}

// Route is route.ts.
var Route = medusahttp.Route{POST: POST}
