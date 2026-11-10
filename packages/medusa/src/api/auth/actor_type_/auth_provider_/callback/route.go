// Package callback mirrors api/auth/[actor_type]/[auth_provider]/callback.
package callback

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	authprovider "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/actor_type_/auth_provider_"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
	"github.com/hitkey-io/medusago/packages/modules/auth/src/services"
)

// GET mirrors GET: the provider validates the callback (the input carries
// no actor_type).
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	actorType := req.Params["actor_type"]
	provider := req.Params["auth_provider"]
	service := req.Scope.Resolve(utilsmodulessdk.AUTH).(*services.AuthModuleService)
	requiresActorAssociation := authutils.IsActorAssociationRequired(authutils.ConfigOf(req.Scope), provider)
	in := authprovider.AuthenticationInputFrom(req)
	in.ActorType = ""
	result := service.ValidateCallback(req.Ctx(), provider, in)
	if result.Success && result.AuthIdentity != nil {
		associated := true
		if requiresActorAssociation {
			am, _ := result.AuthIdentity.Get("app_metadata")
			associated = false
			if r, ok := am.(interface{ Get(string) (any, bool) }); ok {
				v, _ := r.Get(actorType + "_id")
				associated = v != nil && v != ""
			}
		}
		if associated {
			body, err := authutils.GenerateJwtTokenWithChecks(req.Ctx(), req.Scope, result.AuthIdentity, result.MfaChallenge, actorType, provider)
			if err != nil {
				return err
			}
			return res.JSON(body)
		}
	}
	msg := "Authentication failed"
	if !requiresActorAssociation && result.Error != "" {
		msg = result.Error
	}
	return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, msg)
}

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	return GET(req, res)
}

// Route is the route file.
var Route = medusahttp.Route{GET: GET, POST: POST}
