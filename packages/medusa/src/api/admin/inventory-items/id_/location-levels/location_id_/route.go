// Package location_id_ mirrors api/admin/inventory-items/[id]/location-levels/[location_id]/route.ts.
package location_id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/inventory/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	inventoryitems "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/inventory-items"
)

func num(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case string:
		var f float64
		fmt.Sscan(t, &f)
		return f
	}
	return 0
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id, locationID := req.Params["id"], req.Params["location_id"]
	f := dal.NewRecord()
	f.Set("inventory_item_id", id)
	f.Set("location_id", locationID)
	r, err := httputils.QueryOf(req.Scope).Graph(req.Ctx(), query.GraphInput{Entity: "inventory_level", Filters: f, Fields: []string{"id", "reserved_quantity"}}, joiner.Options{})
	if err != nil {
		return err
	}
	if len(r.Data) == 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Inventory Level for Item %s at Location %s not found", id, locationID))
	}
	level, _ := r.Data[0].(*dal.Record)
	levelID, _ := level.Get("id")
	reserved, _ := level.Get("reserved_quantity")
	if num(reserved) > 0 {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Cannot remove Inventory Level %s at Location %s because there are reservations at location", id, locationID))
	}
	lf := dal.NewRecord()
	lf.Set("id", []any{levelID})
	if _, err := workflows.DeleteInventoryLevelsWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteInventoryLevelsInput{Filters: lf}); err != nil {
		return err
	}
	item, err := inventoryitems.Refetch(req, id)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", levelID)
	out.Set("object", "inventory-level")
	out.Set("deleted", true)
	if item != nil {
		out.Set("parent", item)
	}
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	in := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		in.Set(k, v)
	}
	in.Set("inventory_item_id", req.Params["id"])
	in.Set("location_id", req.Params["location_id"])
	if _, err := workflows.UpdateInventoryLevelsWorkflow.Run(req.Ctx(), req.Scope, []*dal.Record{in}); err != nil {
		return err
	}
	return inventoryitems.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST, DELETE: DELETE}
