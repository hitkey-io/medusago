// Package revoke mirrors api/admin/api-keys/[id]/revoke/route.ts.
package revoke

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/api-key/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	apikeys "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	data := dal.NewRecord()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		data.Set(k, v)
	}
	data.Set("revoked_by", req.AuthString("actor_id"))
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	if _, err := workflows.RevokeApiKeysWorkflow.Run(req.Ctx(), req.Scope, workflows.SelectorInput{Selector: selector, Data: data}); err != nil {
		return err
	}
	return apikeys.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
