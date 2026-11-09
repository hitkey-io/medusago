// Package accept mirrors api/admin/invites/accept/route.ts.
package accept

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/invite/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// POST mirrors POST (workflow errors are answered with 401 Unauthorized).
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	if req.AuthString("actor_id") != "" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "The user is already authenticated and cannot accept an invite.")
	}
	var token string
	if req.FilterableFields != nil {
		v, _ := req.FilterableFields.Get("token")
		token, _ = v.(string)
	}
	var authIdentityID any
	if req.AuthContext != nil {
		authIdentityID, _ = req.AuthContext.Get("auth_identity_id")
	}
	user, _ := req.ValidatedBody.(*dal.Record)
	users, err := workflows.AcceptInviteWorkflow.Run(req.Ctx(), req.Scope, workflows.AcceptInviteWorkflowInput{InviteToken: token, AuthIdentityID: authIdentityID, User: user})
	if err != nil {
		out := dal.NewRecord()
		out.Set("message", "Unauthorized")
		return res.Status(401).JSON(out)
	}
	out := dal.NewRecord()
	out.Set("user", users[0])
	return res.Status(200).JSON(out)
}

var authenticate = false

// Route is the route file.
var Route = medusahttp.Route{POST: POST, Authenticate: &authenticate}
