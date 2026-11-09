// Package id_ mirrors api/admin/sales-channels/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	saleschannels "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/sales-channels"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	sc, err := saleschannels.RefetchSalesChannel(req, req.Params["id"], req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if sc == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Sales channel with id: %s not found", req.Params["id"]))
	}
	out := dal.NewRecord()
	out.Set("sales_channel", sc)
	return res.JSON(out)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	existing, err := saleschannels.RefetchSalesChannel(req, id, []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Sales channel with id "%s" not found`, id))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", id)
	if _, err := workflows.UpdateSalesChannelsWorkflow.Run(req.Ctx(), req.Scope, steps.UpdateSalesChannelsStepInput{Selector: selector, Update: body}); err != nil {
		return err
	}
	v, err := saleschannels.RefetchSalesChannel(req, id, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return saleschannels.Respond(res, v)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteSalesChannelsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "sales-channel")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
