// Package auth_provider_ mirrors api/auth/[actor_type]/[auth_provider]/route.ts.
package auth_provider_

import (
	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	authtypes "github.com/hitkey-io/medusago/packages/core/types/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	authutils "github.com/hitkey-io/medusago/packages/medusa/src/api/auth/utils"
	"github.com/hitkey-io/medusago/packages/modules/auth/src/services"
)

// AuthenticationInputFrom builds the AuthenticationInput of a request.
func AuthenticationInputFrom(req *medusahttp.MedusaRequest) authtypes.AuthenticationInput {
	headers := map[string]string{}
	for k, v := range req.Header {
		if len(v) > 0 {
			headers[lower(k)] = v[0]
		}
	}
	return authtypes.AuthenticationInput{
		ActorType: req.Params["actor_type"],
		URL:       req.URL.RequestURI(),
		Headers:   headers,
		Query:     req.Query,
		Body:      req.Body,
		Protocol:  req.Protocol(),
	}
}

func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// GET mirrors GET.
func GET(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse) error {
	actorType := req.Params["actor_type"]
	provider := req.Params["auth_provider"]
	service := req.Scope.Resolve(utilsmodulessdk.AUTH).(*services.AuthModuleService)
	cfg := authutils.ConfigOf(req.Scope)
	requiresActorAssociation := authutils.IsActorAssociationRequired(cfg, provider)

	result := service.Authenticate(req.Ctx(), provider, AuthenticationInputFrom(req))
	if result.Location != "" {
		return res.Status(200).JSON(map[string]any{"location": result.Location})
	}
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
			return res.Status(200).JSON(body)
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
