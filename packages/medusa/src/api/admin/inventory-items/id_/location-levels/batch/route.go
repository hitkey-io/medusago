// Package batch mirrors api/admin/inventory-items/[id]/location-levels/batch/route.ts.
package batch

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
)

func withItem(rs []*dal.Record, id string) []*dal.Record {
	out := make([]*dal.Record, len(rs))
	for i, r := range rs {
		n := dal.NewRecord()
		for _, k := range r.Keys() {
			v, _ := r.Get(k)
			n.Set(k, v)
		}
		n.Set("inventory_item_id", id)
		out[i] = n
	}
	return out
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	id := req.Params["id"]
	c, _ := body.Get("create")
	u, _ := body.Get("update")
	d, _ := body.Get("delete")
	force, _ := body.Get("force")
	del := inventoryitems.Anys(d)
	if del == nil {
		del = []any{}
	}
	f, _ := force.(bool)
	r, err := workflows.BatchInventoryItemLevelsWorkflow.Run(req.Ctx(), req.Scope, workflows.BatchInventoryItemLevelsInput{
		Create: withItem(inventoryitems.Records(c), id),
		Update: withItem(inventoryitems.Records(u), id),
		Delete: del,
		Force:  f,
	})
	if err != nil {
		return err
	}
	return inventoryitems.BatchResponse(res, r, true)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
