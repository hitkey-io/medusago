// Package id_ mirrors api/admin/users/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/user/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/users"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	update := dal.NewRecord()
	update.Set("id", id)
	if body, ok := req.ValidatedBody.(*dal.Record); ok {
		for _, k := range body.Keys() {
			v, _ := body.Get(k)
			update.Set(k, v)
		}
	}
	if _, err := workflows.UpdateUsersWorkflow.Run(req.Ctx(), req.Scope, workflows.UpdateUsersWorkflowInput{Updates: []*dal.Record{update}}); err != nil {
		return err
	}
	user, err := users.RefetchUser(req.Ctx(), id, req.Scope, req.QueryConfig.Fields)
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("user", user)
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if req.AuthString("actor_id") == id {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "A user cannot delete itself")
	}
	if _, err := workflows.RemoveUserAccountWorkflow.Run(req.Ctx(), req.Scope, id); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "user")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file (GET; the remote query variables { id } act as
// the id filter).
var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("user", "user", "User with id: %s was not found"), POST: POST, DELETE: DELETE}
