// Package register mirrors api/auth/[actor_type]/[auth_provider]/register/route.ts.
package register

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	authprovider "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/auth_provider_"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
	"github.com/hitkey-io/medusago/packages/modules/auth/src/services"
)

// POST mirrors POST: the token of a registered identity carries no actor.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	actorType := req.Params["actor_type"]
	provider := req.Params["auth_provider"]
	service := req.Scope.Resolve(utilsmodulessdk.AUTH).(*services.AuthModuleService)
	result := service.Register(req.Ctx(), provider, authprovider.AuthenticationInputFrom(req))
	if result.Success && result.AuthIdentity != nil {
		token, err := authutils.GenerateJwtTokenForAuthIdentity(req.Ctx(), req.Scope, authutils.TokenInput{AuthIdentity: result.AuthIdentity, ActorType: actorType, AuthProvider: provider})
		if err != nil {
			return err
		}
		out := dal.NewRecord()
		out.Set("token", token)
		return res.Status(200).JSON(out)
	}
	msg := result.Error
	if msg == "" {
		msg = "Authentication failed"
	}
	return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, msg)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
