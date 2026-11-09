// Package utils mirrors @medusajs/medusa/src/api/auth/utils.
package utils

import (
	"context"
	"time"

	"github.com/hitkey-io/medusago/packages/core/framework/src/config"
	"github.com/hitkey-io/medusago/packages/core/utils/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Resolver resolves container registrations.
type Resolver interface{ Resolve(key string) any }

// ConfigOf returns the config module of a container.
func ConfigOf(c Resolver) config.ConfigModule {
	cfg, _ := c.Resolve(common.ContainerRegistrationKeys.CONFIG_MODULE).(config.ConfigModule)
	return cfg
}

// TokenInput mirrors the first argument of generateJwtTokenForAuthIdentity.
type TokenInput struct {
	AuthIdentity *dal.Record
	ActorType    string
	AuthProvider string
	// MfaEnabled nil = look up the enabled factors.
	MfaEnabled              *bool
	MfaChallengeCompletedAt any
	// SkipActorType mirrors skipActorType.
	SkipActorType bool
}

func records(v any) []*dal.Record {
	switch t := v.(type) {
	case []*dal.Record:
		return t
	case []any:
		var out []*dal.Record
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				out = append(out, r)
			}
		}
		return out
	}
	return nil
}

// MfaFactorLister is the part of the auth module used to compute mfa_enabled.
type MfaFactorLister interface {
	HasEnabledMfa(ctx context.Context, authIdentityID string) (bool, error)
}

// GenerateJwtTokenForAuthIdentity mirrors generateJwtTokenForAuthIdentity.
func GenerateJwtTokenForAuthIdentity(ctx context.Context, c Resolver, in TokenInput) (string, error) {
	http := ConfigOf(c).ProjectConfig.HTTP
	entityIDKey := in.ActorType + "_id"
	appMetadata, _ := in.AuthIdentity.Get("app_metadata")
	am, _ := appMetadata.(*dal.Record)
	var entityID any
	if !in.SkipActorType && am != nil {
		entityID, _ = am.Get(entityIDKey)
	}
	var providerIdentity *dal.Record
	if in.AuthProvider != "" {
		pis, _ := in.AuthIdentity.Get("provider_identities")
		for _, pi := range records(pis) {
			if p, _ := pi.Get("provider"); p == in.AuthProvider {
				providerIdentity = pi
				break
			}
		}
	}
	identityID, _ := in.AuthIdentity.Get("id")
	mfaEnabled := false
	if in.MfaEnabled != nil {
		mfaEnabled = *in.MfaEnabled
	} else if idStr, ok := identityID.(string); ok && idStr != "" {
		if svc, ok := c.Resolve("auth").(MfaFactorLister); ok {
			enabled, err := svc.HasEnabledMfa(ctx, idStr)
			if err != nil {
				return "", err
			}
			mfaEnabled = enabled
		}
	}

	payload := dal.NewRecord()
	if s, ok := entityID.(string); ok {
		payload.Set("actor_id", s)
	} else {
		payload.Set("actor_id", "")
	}
	payload.Set("actor_type", in.ActorType)
	if identityID == nil {
		identityID = ""
	}
	payload.Set("auth_identity_id", identityID)
	if in.AuthProvider != "" {
		payload.Set("auth_provider", in.AuthProvider)
	}
	payload.Set("mfa_enabled", mfaEnabled)
	payload.Set("mfa_challenge_completed_at", isoOrNil(in.MfaChallengeCompletedAt))
	meta := dal.NewRecord()
	if am != nil {
		for _, k := range am.Keys() {
			v, _ := am.Get(k)
			meta.Set(k, v)
		}
	}
	// [entityIdKey]: entityId — undefined is dropped by JSON.stringify.
	if entityID != nil {
		meta.Set(entityIDKey, entityID)
	} else {
		meta.Delete(entityIDKey)
	}
	payload.Set("app_metadata", meta)
	var userMetadata any = dal.NewRecord()
	if providerIdentity != nil {
		if um, ok := providerIdentity.Get("user_metadata"); ok && um != nil {
			userMetadata = um
		}
	}
	payload.Set("user_metadata", userMetadata)
	return auth.GenerateJwtToken(payload, http.JwtSecret, http.JwtExpiresIn, auth.JwtOptions{})
}

func isoOrNil(v any) any {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if t == "" {
			return nil
		}
		if p, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return dal.FormatJSDate(p)
		}
		return t
	case time.Time:
		return dal.FormatJSDate(t)
	}
	return v
}

// GenerateJwtTokenWithChecks mirrors generateJwtTokenWithChecks: the
// response body of a successful authentication.
func GenerateJwtTokenWithChecks(ctx context.Context, c Resolver, authIdentity *dal.Record, mfaChallenge *dal.Record, actorType, authProvider string) (*dal.Record, error) {
	mfa := mfaChallenge != nil
	if mfa {
		token, err := GenerateJwtTokenForAuthIdentity(ctx, c, TokenInput{AuthIdentity: authIdentity, ActorType: actorType, AuthProvider: authProvider, MfaEnabled: &mfa, SkipActorType: true})
		if err != nil {
			return nil, err
		}
		out := dal.NewRecord()
		out.Set("mfa_required", true)
		out.Set("mfa_challenge", mfaChallenge)
		out.Set("token", token)
		return out, nil
	}
	token, err := GenerateJwtTokenForAuthIdentity(ctx, c, TokenInput{AuthIdentity: authIdentity, ActorType: actorType, AuthProvider: authProvider, MfaEnabled: &mfa})
	if err != nil {
		return nil, err
	}
	out := dal.NewRecord()
	out.Set("token", token)
	return out, nil
}
