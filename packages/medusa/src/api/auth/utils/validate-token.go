package utils

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

type resetTokenConsumer interface {
	ConsumePasswordResetToken(ctx context.Context, jti, provider, entityID string) (*dal.Record, error)
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
}

// ValidateToken mirrors validateToken: the bearer token must be a reset
// token (purpose "reset" with a jti) for the route's actor type; it is
// consumed and the auth context set from the provider identity.
func ValidateToken() medusahttp.MiddlewareFunction {
	return func(req *medusahttp.MedusaRequest, res *medusahttp.MedusaResponse, next medusahttp.NextFunction) error {
		actorType, provider := req.Params["actor_type"], req.Params["auth_provider"]
		invalid := common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "Invalid token")
		token := middlewares.GetAuthContextFromJwtToken(req.Header.Get("Authorization"), ConfigOf(req.Scope).ProjectConfig.HTTP.JwtSecret, []string{"bearer"}, []string{actorType})
		get := func(k string) any {
			if token == nil {
				return nil
			}
			v, _ := token.Get(k)
			return v
		}
		entityID, _ := get("entity_id").(string)
		if token == nil || entityID == "" {
			next(invalid)
			return nil
		}
		jti, _ := get("jti").(string)
		if get("purpose") != "reset" || jti == "" {
			next(invalid)
			return nil
		}
		svc, _ := req.Scope.Resolve(utilsmodulessdk.AUTH).(resetTokenConsumer)
		if _, err := svc.ConsumePasswordResetToken(req.Ctx(), jti, provider, entityID); err != nil {
			next(invalid)
			return nil
		}
		f := dal.NewRecord()
		f.Set("entity_id", entityID)
		f.Set("provider", provider)
		identities, err := svc.ListBySuffix(req.Ctx(), "ProviderIdentities", f, utilsmodulessdk.FindConfig{Select: []string{"provider_metadata", "auth_identity_id", "entity_id", "user_metadata"}})
		if err != nil || len(identities) == 0 {
			next(invalid)
			return nil
		}
		identity := identities[0]
		authIdentityID, _ := identity.Get("auth_identity_id")
		actorID, _ := identity.Get("entity_id")
		userMetadata, _ := identity.Get("user_metadata")
		if userMetadata == nil {
			userMetadata = dal.NewRecord()
		}
		ac := dal.NewRecord()
		ac.Set("actor_type", actorType)
		ac.Set("auth_identity_id", authIdentityID)
		ac.Set("actor_id", actorID)
		ac.Set("app_metadata", dal.NewRecord())
		ac.Set("user_metadata", userMetadata)
		req.AuthContext = ac
		next(nil)
		return nil
	}
}
