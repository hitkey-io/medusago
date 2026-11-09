package utils

import (
	"fmt"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// ValidateScopeProviderAssociation mirrors validateScopeProviderAssociation.
func ValidateScopeProviderAssociation(actorType string) medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, _ *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		provider := req.Params["auth_provider"]
		actor := actorType
		if actor == "" {
			actor = req.Params["actor_type"]
		}
		if !IsAuthProviderAllowedForActor(ConfigOf(req.Scope), actor, provider) {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("The actor type %s is not allowed to use the auth provider %s", actor, provider))
		}
		next(nil)
		return nil
	}
}
