// Package id_ mirrors api/admin/campaigns/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns"
)

func respond(res *medusahttp.MedusaResponse, campaign any) error {
	out := dal.NewRecord()
	out.Set("campaign", campaign)
	return res.Status(200).JSON(out)
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	campaign, err := campaigns.RefetchCampaign(req.Ctx(), id, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	if campaign == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Campaign with id: %s was not found", id))
	}
	return respond(res, campaign)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	existing, err := campaigns.RefetchCampaign(req.Ctx(), id, req.Scope, []string{"id"})
	if err != nil {
		return err
	}
	if existing == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Campaign with id \"%s\" not found", id))
	}
	body, _ := req.ValidatedBody.(*dal.Record)
	rest, additional := campaigns.SplitAdditionalData(body)
	data := dal.NewRecord()
	data.Set("id", id)
	for _, k := range rest.Keys() {
		v, _ := rest.Get(k)
		data.Set(k, v)
	}
	if _, err := workflows.UpdateCampaignsWorkflow.Run(req.Ctx(), req.Scope, workflows.CampaignsInput{CampaignsData: []*dal.Record{data}, AdditionalData: additional}); err != nil {
		return err
	}
	campaign, err := campaigns.RefetchCampaign(req.Ctx(), id, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	return respond(res, campaign)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteCampaignsWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "campaign")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST, DELETE: DELETE}
