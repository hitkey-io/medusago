// Package services mirrors @medusajs/auth/src/services.
package services

import (
	"context"
	"fmt"

	authtypes "github.com/hitkey-io/medusago/packages/core/types/src/auth"
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/auth/src/models"
)

// AuthModuleService mirrors AuthModuleService.
type AuthModuleService struct {
	*utilsmodulessdk.MedusaService
	providers map[string]authtypes.AuthProvider
	// identifiers mirrors auth_providers_identifier (registration order).
	identifiers []string
}

// NewAuthModuleService builds the service with the configured providers.
func NewAuthModuleService(opts modulessdk.ServiceOptions, joinerConfig *joiner.ModuleJoinerConfig) (*AuthModuleService, error) {
	s := &AuthModuleService{
		MedusaService: utilsmodulessdk.NewMedusaService(opts.DB, models.All, joinerConfig),
		providers:     map[string]authtypes.AuthProvider{},
	}
	for _, p := range opts.Providers {
		prov, ok := p.Instance.(authtypes.AuthProvider)
		if !ok {
			continue
		}
		id := p.ID
		if id == "" {
			id = prov.Identifier()
		}
		s.providers[id] = prov
		s.identifiers = append(s.identifiers, id)
	}
	return s, nil
}

func (s *AuthModuleService) retrieveProvider(id string) (authtypes.AuthProvider, error) {
	p, ok := s.providers[id]
	if !ok {
		return nil, fmt.Errorf("\nUnable to retrieve the auth provider with id: %s\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.", id)
	}
	return p, nil
}

// ListAuthProviders mirrors listAuthProviders (ids nil = no filter).
func (s *AuthModuleService) ListAuthProviders(ids []string) []*dal.Record {
	out := []*dal.Record{}
	for _, id := range s.identifiers {
		if ids != nil {
			found := false
			for _, x := range ids {
				if x == id {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		p, err := s.retrieveProvider(id)
		if err != nil {
			continue
		}
		r := dal.NewRecord()
		r.Set("id", id)
		identifier := p.Identifier()
		if identifier == "" {
			identifier = id
		}
		r.Set("identifier", identifier)
		dn := p.DisplayName()
		if dn == "" {
			dn = id
		}
		r.Set("display_name", dn)
		if p.IsRedirect() {
			r.Set("flow", "redirect")
		} else {
			r.Set("flow", "credentials")
		}
		out = append(out, r)
	}
	return out
}

// Authenticate mirrors authenticate: provider errors become an
// unsuccessful response.
func (s *AuthModuleService) Authenticate(ctx context.Context, provider string, in authtypes.AuthenticationInput) authtypes.AuthenticationResponse {
	p, err := s.retrieveProvider(provider)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	res, err := p.Authenticate(ctx, in, s.identityService(provider))
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	res, err = s.applyMfaRequirement(ctx, res, provider)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	return res
}

// Register mirrors register.
func (s *AuthModuleService) Register(ctx context.Context, provider string, in authtypes.AuthenticationInput) authtypes.AuthenticationResponse {
	p, err := s.retrieveProvider(provider)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	res, err := p.Register(ctx, in, s.identityService(provider))
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	return res
}

// UpdateProvider mirrors updateProvider.
func (s *AuthModuleService) UpdateProvider(ctx context.Context, provider string, data *dal.Record) authtypes.AuthenticationResponse {
	p, err := s.retrieveProvider(provider)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	res, err := p.Update(ctx, data, s.identityService(provider))
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	return res
}

// ValidateCallback mirrors validateCallback.
func (s *AuthModuleService) ValidateCallback(ctx context.Context, provider string, in authtypes.AuthenticationInput) authtypes.AuthenticationResponse {
	p, err := s.retrieveProvider(provider)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	res, err := p.ValidateCallback(ctx, in, s.identityService(provider))
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	res, err = s.applyMfaRequirement(ctx, res, provider)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}
	}
	return res
}

// applyMfaRequirement mirrors applyMfaRequirement_: an identity with
// verifiable MFA methods gets a challenge instead of a session.
func (s *AuthModuleService) applyMfaRequirement(ctx context.Context, res authtypes.AuthenticationResponse, authProvider string) (authtypes.AuthenticationResponse, error) {
	if !res.Success || res.AuthIdentity == nil || res.Location != "" {
		return res, nil
	}
	id, _ := res.AuthIdentity.Get("id")
	methods, err := s.availableMfaChallengeMethods(ctx, id)
	if err != nil {
		return res, err
	}
	if len(methods) == 0 {
		return res, nil
	}
	var provider any
	if authProvider != "" {
		provider = authProvider
	}
	challenge, err := s.createAuthMfaChallenge(ctx, id, provider, nil)
	if err != nil {
		return res, err
	}
	return authtypes.AuthenticationResponse{Success: true, MfaChallenge: challenge, AuthIdentity: res.AuthIdentity}, nil
}

// identityService mirrors getAuthIdentityProviderService.
func (s *AuthModuleService) identityService(provider string) authtypes.AuthIdentityProviderService {
	return &identityProviderService{s: s, provider: provider}
}

type identityProviderService struct {
	s        *AuthModuleService
	provider string
}

func (i *identityProviderService) find(ctx context.Context, entityID string) ([]*dal.Record, error) {
	pi := dal.NewRecord()
	pi.Set("entity_id", entityID)
	pi.Set("provider", i.provider)
	f := dal.NewRecord()
	f.Set("provider_identities", pi)
	list, err := i.s.Internal("AuthIdentity").List(ctx, f, utilsmodulessdk.FindConfig{Relations: []string{"provider_identities"}})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`AuthIdentity with entity_id "%s" not found`, entityID))
	}
	if len(list) > 1 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`Multiple authIdentities found for entity_id "%s"`, entityID))
	}
	return list, nil
}

func (i *identityProviderService) Retrieve(ctx context.Context, entityID string) (*dal.Record, error) {
	list, err := i.find(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return list[0], nil
}

func (i *identityProviderService) Create(ctx context.Context, data authtypes.CreateAuthIdentityInput) (*dal.Record, error) {
	pi := dal.NewRecord()
	pi.Set("entity_id", data.EntityID)
	if data.ProviderMetadata != nil {
		pi.Set("provider_metadata", data.ProviderMetadata)
	}
	if data.UserMetadata != nil {
		pi.Set("user_metadata", data.UserMetadata)
	}
	pi.Set("provider", i.provider)
	in := dal.NewRecord()
	in.Set("provider_identities", []any{pi})
	created, err := i.s.Internal("AuthIdentity").Create(ctx, []*dal.Record{in})
	if err != nil {
		return nil, err
	}
	return created[0], nil
}

func (i *identityProviderService) Update(ctx context.Context, entityID string, data *dal.Record) (*dal.Record, error) {
	list, err := i.find(ctx, entityID)
	if err != nil {
		return nil, err
	}
	identity := list[0]
	var target *dal.Record
	var others []any
	v, _ := identity.Get("provider_identities")
	for _, x := range toRecords(v) {
		p, _ := x.Get("provider")
		e, _ := x.Get("entity_id")
		if target == nil && p == i.provider && e == entityID {
			target = x
			continue
		}
		others = append(others, x)
	}
	if target == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`ProviderIdentity with entity_id "%s" not found`, entityID))
	}
	upd := dal.NewRecord()
	id, _ := target.Get("id")
	upd.Set("id", id)
	for _, k := range data.Keys() {
		x, _ := data.Get(k)
		upd.Set(k, x)
	}
	res, err := i.s.Internal("ProviderIdentity").Update(ctx, upd)
	if err != nil {
		return nil, err
	}
	updated, _ := res.(*dal.Record)
	identity.Set("provider_identities", append(others, updated))
	return identity, nil
}

// SetState mirrors setState: the state is kept in the cache (20 minutes by
// default).
func (i *identityProviderService) SetState(ctx context.Context, key string, value *dal.Record, ttlSeconds int) error {
	if ttlSeconds == 0 {
		ttlSeconds = 1200
	}
	cacheSet(key, value, ttlSeconds)
	return nil
}

// GetState mirrors getState: a state is single use (consumed on read).
func (i *identityProviderService) GetState(ctx context.Context, key string) (*dal.Record, error) {
	v := cacheGet(key)
	if v != nil {
		cacheInvalidate(key)
	}
	return v, nil
}

func toRecords(v any) []*dal.Record {
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

// HasEnabledMfa mirrors the listAuthMfa({ status: "enabled" }) lookup of
// generateJwtTokenForAuthIdentity.
func (s *AuthModuleService) HasEnabledMfa(ctx context.Context, authIdentityID string) (bool, error) {
	f := dal.NewRecord()
	f.Set("auth_identity_id", authIdentityID)
	f.Set("status", "enabled")
	list, err := s.Internal("AuthMfaFactor").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
	return len(list) > 0, err
}
