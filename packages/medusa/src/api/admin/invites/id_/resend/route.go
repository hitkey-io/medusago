// Package resend mirrors api/admin/invites/[id]/resend/route.ts.
package resend

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/invite/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/admin/invites"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	result, err := workflows.RefreshInviteTokensWorkflow.Run(req.Ctx(), req.Scope, []any{req.Params["id"]})
	if err != nil {
		return err
	}
	id, _ := result[0].Get("id")
	invite, err := invites.RefetchInvite(req.Ctx(), req, fmt.Sprint(id))
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("invite", invite)
	return res.Status(200).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
