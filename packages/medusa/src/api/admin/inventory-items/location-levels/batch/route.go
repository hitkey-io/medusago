// Package batch mirrors api/admin/inventory-items/location-levels/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	c, _ := body.Get("create")
	u, _ := body.Get("update")
	d, _ := body.Get("delete")
	force, _ := body.Get("force")
	f, _ := force.(bool)
	r, err := workflows.BatchInventoryItemLevelsWorkflow.Run(req.Ctx(), req.Scope, workflows.BatchInventoryItemLevelsInput{
		Create: inventoryitems.Records(c),
		Update: inventoryitems.Records(u),
		Delete: inventoryitems.Anys(d),
		Force:  f,
	})
	if err != nil {
		return err
	}
	return inventoryitems.BatchResponse(res, r, false)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
