// Package id_ mirrors api/admin/return-reasons/[id]/route.ts.
package id_

import (
	"fmt"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/return-reason/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/return-reason/workflows"
	returnreasons "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/return-reasons"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	r, err := httputils.RefetchEntity(req.Ctx(), req.Scope, "return_reason", req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if r == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Return reason with id: %s was not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("return_reason", r)
	return res.JSON(out)
}

// Route is the route file.
// POST mirrors POST (result[0].id of an empty result throws upstream).
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	result, err := workflows.UpdateReturnReasonsWorkflow.Run(req.Ctx(), req.Scope, steps.UpdateReturnReasonsStepInput{Selector: selector, Update: body})
	if err != nil {
		return err
	}
	if len(result) == 0 {
		return fmt.Errorf("Cannot read properties of undefined (reading 'id')")
	}
	id, _ := result[0].Get("id")
	return returnreasons.RespondByID(req, res, id)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteReturnReasonsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "return_reason")
	out.Set("deleted", true)
	return res.JSON(out)
}

var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
