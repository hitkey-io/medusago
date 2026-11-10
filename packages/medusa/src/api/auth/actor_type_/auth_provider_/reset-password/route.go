// Package resetpassword mirrors api/auth/[actor_type]/[auth_provider]/reset-password.
package resetpassword

import (
	"fmt"

	authworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/workflows"
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
)

// POST mirrors POST: errors are swallowed (throwOnError: false) so the
// response does not reveal whether the identity exists.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	body, _ := req.ValidatedBody.(*dal.Record)
	identifier, _ := body.Get("identifier")
	metadata, _ := body.Get("metadata")
	meta, _ := metadata.(*dal.Record)
	_, _ = authworkflows.GenerateResetPasswordTokenWorkflow.Run(req.Ctx(), req.Scope, authworkflows.GenerateResetPasswordTokenInput{
		EntityID: fmt.Sprint(identifier), ActorType: req.Params["actor_type"], Provider: req.Params["auth_provider"],
		Secret: authutils.ConfigOf(req.Scope).ProjectConfig.HTTP.JwtSecret, Metadata: meta,
	})
	res.SendStatus(201)
	return nil
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
