// Package id_ mirrors api/admin/invites/[id]/route.ts.
package id_

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/invite/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/invites"
)

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	invite, err := invites.RefetchInvite(req.Ctx(), req, id)
	if err != nil {
		return err
	}
	if invite == nil {
		return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Invite with id: %s was not found", id))
	}
	out := dal.NewRecord()
	out.Set("invite", invite)
	return res.Status(200).JSON(out)
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteInvitesWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "invite")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, DELETE: DELETE}
