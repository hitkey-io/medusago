// Package id_ mirrors api/admin/payment-collections/[id].
package id_

import (
	orderworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/order/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// DELETE mirrors DELETE /admin/payment-collections/:id.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := orderworkflows.DeleteOrderPaymentCollections.Run(req.Ctx(), req.Scope, id); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "payment-collection")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is route.ts.
var Route = medusahttp.Route{DELETE: DELETE}
