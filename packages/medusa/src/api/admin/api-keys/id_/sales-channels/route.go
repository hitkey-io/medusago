// Package saleschannels mirrors api/admin/api-keys/[id]/sales-channels/route.ts.
package saleschannels

import (
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/api-key/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	apikeys "github.com/hitkey-io/medusago/packages/medusa/src/api/admin/api-keys"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	key, err := apikeys.RefetchApiKey(req, req.Params["id"], []string{"id", "type"})
	if err != nil {
		return err
	}
	r, ok := key.(*dal.Record)
	if !ok || r == nil {
		// apiKey.type of undefined throws upstream
		return fmt.Errorf("Cannot read properties of undefined (reading 'type')")
	}
	typ, _ := r.Get("type")
	if typ != "publishable" {
		return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Sales channels can only be associated with publishable API keys")
	}
	add, _ := body.Get("add")
	remove, _ := body.Get("remove")
	if _, err := workflows.LinkSalesChannelsToApiKeyWorkflow.Run(req.Ctx(), req.Scope, workflows.LinkInput{ID: req.Params["id"], Add: apikeys.Strings(add), Remove: apikeys.Strings(remove)}); err != nil {
		return err
	}
	return apikeys.Respond(req, res, req.Params["id"])
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
