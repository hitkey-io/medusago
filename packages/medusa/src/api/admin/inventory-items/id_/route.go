// Package id_ mirrors api/admin/inventory-items/[id]/route.ts.
package id_

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// RefetchInventoryItem mirrors refetchInventoryItem (a paginated remote
// query: skip 0, take 1).
func RefetchInventoryItem(req *medusahttp.MedusaRequest, id string, fields []string) (any, error) {
	filters := dal.NewRecord()
	filters.Set("id", id)
	vars := dal.NewRecord()
	vars.Set("filters", filters)
	vars.Set("skip", float64(0))
	vars.Set("take", float64(1))
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, "inventory_item", vars, fields)
	if err != nil {
		return nil, err
	}
	if len(r.Data) == 0 {
		return nil, nil
	}
	return r.Data[0], nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	item, err := RefetchInventoryItem(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if item == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Inventory item with id: %s was not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("inventory_item", item)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET}
