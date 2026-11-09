// Package auth mirrors @medusajs/types/src/auth.
package auth

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// AuthenticationInput mirrors AuthenticationInput.
type AuthenticationInput struct {
	ActorType string
	URL       string
	Headers   map[string]string
	Query     *dal.Record
	Body      any
	Protocol  string
}

// BodyString returns a string field of the body (JS: body?.[key]) and
// whether it is a non-empty string.
func (in AuthenticationInput) BodyString(key string) (string, bool) {
	r, ok := in.Body.(*dal.Record)
	if !ok || r == nil {
		return "", false
	}
	v, _ := r.Get(key)
	s, ok := v.(string)
	return s, ok && s != ""
}

// AuthenticationResponse mirrors AuthenticationResponse.
type AuthenticationResponse struct {
	Success      bool
	AuthIdentity *dal.Record
	Error        string
	Location     string
	MfaChallenge *dal.Record
}

// CreateAuthIdentityInput is the input of AuthIdentityProviderService.Create.
type CreateAuthIdentityInput struct {
	EntityID         string
	ProviderMetadata *dal.Record
	UserMetadata     *dal.Record
}

// AuthIdentityProviderService mirrors AuthIdentityProviderService.
type AuthIdentityProviderService interface {
	Retrieve(ctx context.Context, entityID string) (*dal.Record, error)
	Create(ctx context.Context, data CreateAuthIdentityInput) (*dal.Record, error)
	Update(ctx context.Context, entityID string, data *dal.Record) (*dal.Record, error)
}

// AuthProvider mirrors IAuthProvider (AbstractAuthModuleProvider).
type AuthProvider interface {
	Identifier() string
	DisplayName() string
	// IsRedirect reports whether validateCallback is overridden (flow
	// "redirect").
	IsRedirect() bool
	Authenticate(ctx context.Context, data AuthenticationInput, svc AuthIdentityProviderService) (AuthenticationResponse, error)
	Register(ctx context.Context, data AuthenticationInput, svc AuthIdentityProviderService) (AuthenticationResponse, error)
	Update(ctx context.Context, data *dal.Record, svc AuthIdentityProviderService) (AuthenticationResponse, error)
	ValidateCallback(ctx context.Context, data AuthenticationInput, svc AuthIdentityProviderService) (AuthenticationResponse, error)
}
