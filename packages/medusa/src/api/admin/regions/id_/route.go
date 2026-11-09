// Package id_ mirrors api/admin/regions/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/region/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/regions"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	region, err := regions.RefetchRegion(req.Ctx(), req.Scope, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if region == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Region with id: %s not found", req.Params["id"]))
	}
	return regions.Respond(res, region)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	existing, err := regions.RefetchRegion(req.Ctx(), req.Scope, req.Params["id"], []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Region with id "%s" not found`, req.Params["id"]))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	result, err := workflows.UpdateRegionsWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateRegionsInput{Selector: selector, Update: body})
	if err != nil {
		return err
	}
	var id any
	if len(result) > 0 {
		id, _ = result[0].Get("id")
	}
	region, err := regions.RefetchRegion(req.Ctx(), req.Scope, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return regions.Respond(res, region)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteRegionsWorkflow.Run(req.Ctx(), req.Scope, workflows.DeleteRegionsInput{IDs: []any{id}}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "region")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
