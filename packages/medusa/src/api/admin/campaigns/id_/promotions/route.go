// Package promotions mirrors api/admin/campaigns/[id]/promotions/route.ts.
package promotions

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/promotion/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/campaigns"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	in := steps.LinkInput{ID: id}
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		a, _ := body.Get("add")
		r, _ := body.Get("remove")
		in.Add, _ = a.([]any)
		in.Remove, _ = r.([]any)
	}
	if _, err := workflows.AddOrRemoveCampaignPromotionsWorkflow.Run(req.Ctx(), req.Scope, in); err != nil {
		return err
	}
	campaign, err := campaigns.RefetchCampaign(req.Ctx(), id, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("campaign", campaign)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
