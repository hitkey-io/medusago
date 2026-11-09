// Package refresh mirrors api/auth/token/refresh/route.ts.
package refresh

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
	"github.com/hitkey-io/medusago/packages/modules/auth/src/services"
)

// POST mirrors POST.
func POST(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	service := req.Scope.Resolve(utilsmodulessdk.AUTH).(*services.AuthModuleService)
	provider := req.AuthString("auth_provider")
	if provider == "" {
		return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "The auth provider is not set while refreshing token")
	}
	identityID := req.AuthString("auth_identity_id")
	identity, err := service.Internal("AuthIdentity").Retrieve(req.Ctx(), identityID, utilsmodulessdk.FindConfig{Relations: []string{"provider_identities"}})
	if err != nil {
		return err
	}
	if req.AuthString("actor_id") != "" {
		mfa, _ := req.AuthContext.Get("mfa_enabled")
		mfaEnabled, _ := mfa.(bool)
		completed, _ := req.AuthContext.Get("mfa_challenge_completed_at")
		token, err := authutils.GenerateJwtTokenForAuthIdentity(req.Ctx(), req.Scope, authutils.TokenInput{
			AuthIdentity:            identity,
			ActorType:               req.AuthString("actor_type"),
			AuthProvider:            provider,
			MfaEnabled:              &mfaEnabled,
			MfaChallengeCompletedAt: completed,
		})
		if err != nil {
			return err
		}
		out := dal.NewRecord()
		out.Set("token", token)
		return res.JSON(out)
	}
	body, err := authutils.GenerateJwtTokenWithChecks(req.Ctx(), req.Scope, identity, nil, req.AuthString("actor_type"), provider)
	if err != nil {
		return err
	}
	return res.JSON(body)
}

// Route is the route file.
var Route = medusahttp.Route{POST: POST}
