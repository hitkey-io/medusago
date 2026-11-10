// Package id_ mirrors api/admin/order-edits/[id].
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// DELETE mirrors DELETE /admin/order-edits/:id.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := workflows.CancelBeginOrderEditWorkflow.Run(req.Ctx(), req.Scope, any(req.Params["id"])); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", req.Params["id"])
	out.Set("object", "order-edit")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{DELETE: DELETE}
