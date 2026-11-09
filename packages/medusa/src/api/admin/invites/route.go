package invites

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/invite/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	result, err := workflows.CreateInvitesWorkflow.Run(req.Ctx(), req.Scope, workflows.CreateInvitesWorkflowInput{Invites: []*dal.Record{body}})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	invite, err := RefetchInvite(req.Ctx(), req, fmt.Sprint(id))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("invite", invite)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: httputils.RemoteQueryListRoute("invite", "invites"), POST: POST}
