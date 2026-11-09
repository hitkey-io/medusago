package utils

import (
	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
)

// GetAllowedAuthProvidersForActor mirrors getAllowedAuthProvidersForActor
// (nil = no restriction).
func GetAllowedAuthProvidersForActor(cfg config.ConfigModule, actorType string) []string {
	return cfg.ProjectConfig.HTTP.AuthMethodsPerActor[actorType]
}

// IsAuthProviderAllowedForActor mirrors isAuthProviderAllowedForActor.
func IsAuthProviderAllowedForActor(cfg config.ConfigModule, actorType, provider string) bool {
	allowed := GetAllowedAuthProvidersForActor(cfg, actorType)
	if allowed == nil {
		return true
	}
	for _, p := range allowed {
		if p == provider {
			return true
		}
	}
	return false
}

// IsActorAssociationRequired mirrors isActorAssociationRequired.
func IsActorAssociationRequired(cfg config.ConfigModule, provider string) bool {
	return cfg.ProjectConfig.HTTP.RequireActorAssociation[provider]
}
