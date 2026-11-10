// Package request mirrors api/auth/verification/request.
package request

import (
	authworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/medusa/src/api/auth/mfa"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	b := mfa.Body(req)
	get := func(k string) any { v, _ := b.Get(k); return v }
	r, err := authworkflows.RequestVerificationWorkflow.Run(req.Ctx(), req.Scope, authworkflows.RequestVerificationInput{
		AuthIdentityID: mfa.AuthIdentityID(req), EntityID: get("entity_id"), EntityType: get("entity_type"), CodeProvider: get("code_provider"), Metadata: get("metadata"),
	})
	if err != nil {
		return err
	}
	out := dal.NewRecord()
	out.Set("verification", r)
	return res.Status(201).JSON(out)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
