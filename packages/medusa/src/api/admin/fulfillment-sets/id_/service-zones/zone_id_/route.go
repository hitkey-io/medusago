// Package zone_id_ mirrors api/admin/fulfillment-sets/[id]/service-zones/[zone_id]/route.ts.
package zone_id_

import (
	"fmt"

	fulfillmentsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/fulfillment/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	fulfillmentsets "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/fulfillment-sets"
)

func firstByID(req *medusahttp.MedusaRequest, entry string, id any) (any, error) {
	vars := dal.NewRecord()
	vars.Set("id", id)
	r, err := httputils.RemoteQueryList(req.Ctx(), req.Scope, entry, vars, req.QueryConfig.Fields)
	if err != nil || len(r.Data) == 0 {
		return nil, err
	}
	return r.Data[0], nil
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	zone, err := firstByID(req, "service_zones", req.Params["zone_id"])
	if err != nil {
		return err
	}
	if zone == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Service zone with id: %s not found", req.Params["zone_id"]))
	}
	out := dal.NewRecord()
	out.Set("service_zone", zone)
	return res.Status(200).JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if _, err := fulfillmentsets.RetrieveSetWithZones(req, req.Params["id"], req.Params["zone_id"]); err != nil {
		return err
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["zone_id"])
	if _, err := workflows.UpdateServiceZonesWorkflow.Run(req.Ctx(), req.Scope, fulfillmentsteps.SelectorUpdateInput{Selector: selector, Update: body}); err != nil {
		return err
	}
	set, err := firstByID(req, "fulfillment_sets", req.Params["id"])
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	if set != nil {
		out.Set("fulfillment_set", set)
	}
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE (parent is the fulfillment set loaded before the
// deletion, serialized as the module returns it).
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	set, err := fulfillmentsets.RetrieveSetWithZones(req, req.Params["id"], req.Params["zone_id"])
	if err != nil {
		return err
	}
	if _, err := workflows.DeleteServiceZonesWorkflow.Run(req.Ctx(), req.Scope, []any{req.Params["zone_id"]}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", req.Params["zone_id"])
	out.Set("object", "service_zone")
	out.Set("deleted", true)
	out.Set("parent", set)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
