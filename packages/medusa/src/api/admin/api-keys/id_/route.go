// Package id_ mirrors api/admin/api-keys/[id]/route.ts.
package id_

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/api-key/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	httputils "github.com/hitkey-io/medusago/packages/core/framework/src/http/utils"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	apikeys "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	selector := dal.NewRecord()
	selector.Set("id", req.Params["id"])
	if _, err := workflows.UpdateApiKeysWorkflow.Run(req.Ctx(), req.Scope, workflows.SelectorInput{Selector: selector, Data: body}); err != nil {
		return err
	}
	return apikeys.Respond(req, res, req.Params["id"])
}

// DELETE mirrors DELETE.
func DELETE(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	id := req.Params["id"]
	if _, err := workflows.DeleteApiKeysWorkflow.Run(req.Ctx(), req.Scope, []any{id}); err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("id", id)
	out.Set("object", "api_key")
	out.Set("deleted", true)
	return res.Status(200).JSON(out)
}

var Route = medusahttp.Route{GET: httputils.RemoteQueryRetrieveRoute("api_key", "api_key", "API Key with id: %s was not found"), POST: POST, DELETE: DELETE}
