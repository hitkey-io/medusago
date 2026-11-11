// Package services mirrors @medusajs/auth-github/src/services.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	authtypes "github.com/hitkey-io/medusago/packages/core/types/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

const (
	// Identifier is GithubAuthService.identifier.
	Identifier = "github"
	// DisplayName is GithubAuthService.DISPLAY_NAME.
	DisplayName = "Github Authentication"
)

var _ authtypes.AuthProvider = (*GithubAuthService)(nil)

// GithubAuthService mirrors GithubAuthService.
type GithubAuthService struct {
	clientID     any
	clientSecret any
	callbackURL  any
	logger       *slog.Logger
}

// ValidateOptions mirrors validateOptions.
func ValidateOptions(options map[string]any) error {
	if !truthy(options["clientId"], options["clientId"] != nil) {
		return errors.New("Github clientId is required")
	}
	if !truthy(options["clientSecret"], options["clientSecret"] != nil) {
		return errors.New("Github clientSecret is required")
	}
	if !truthy(options["callbackUrl"], options["callbackUrl"] != nil) {
		return errors.New("Github callbackUrl is required")
	}
	return nil
}

// NewGithubAuthService mirrors the constructor (options clientId,
// clientSecret, callbackUrl).
func NewGithubAuthService(options map[string]any, logger *slog.Logger) (*GithubAuthService, error) {
	if err := ValidateOptions(options); err != nil {
		return nil, err
	}
	return &GithubAuthService{clientID: options["clientId"], clientSecret: options["clientSecret"], callbackURL: options["callbackUrl"], logger: logger}, nil
}

func (s *GithubAuthService) Identifier() string  { return Identifier }
func (s *GithubAuthService) DisplayName() string { return DisplayName }
func (s *GithubAuthService) IsRedirect() bool    { return true }

// Register mirrors register.
func (s *GithubAuthService) Register(context.Context, authtypes.AuthenticationInput, authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	return authtypes.AuthenticationResponse{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Github does not support registration. Use method `authenticate` instead.")
}

// Update mirrors AbstractAuthModuleProvider.update.
func (s *GithubAuthService) Update(context.Context, *dal.Record, authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	return authtypes.AuthenticationResponse{}, fmt.Errorf("Method 'update' not implemented for provider %s", Identifier)
}

// queryError mirrors the `query.error` check of authenticate and
// validateCallback.
func queryError(query *dal.Record) (string, bool) {
	e, ok := get(query, "error")
	if !truthy(e, ok) {
		return "", false
	}
	desc, dok := get(query, "error_description")
	uri, uok := get(query, "error_uri")
	return fmt.Sprintf("%s, read more at: %s", jsString(desc, dok), jsString(uri, uok)), true
}

// Authenticate mirrors authenticate: a state is stored and the client is
// redirected to GitHub.
func (s *GithubAuthService) Authenticate(ctx context.Context, req authtypes.AuthenticationInput, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	if msg, ok := queryError(req.Query); ok {
		return authtypes.AuthenticationResponse{Error: msg}, nil
	}
	stateKey := randomHex(32)
	callback, ok := get(req.Body, "callback_url")
	if !ok || callback == nil {
		callback = s.callbackURL
	}
	state := dal.NewRecord()
	state.Set("callback_url", callback)
	if err := svc.SetState(ctx, stateKey, state, 0); err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	return s.getRedirect(jsString(s.clientID, true), jsString(callback, true), stateKey), nil
}

// ValidateCallback mirrors validateCallback: the code is exchanged for a
// token and the GitHub user is upserted as the auth identity.
func (s *GithubAuthService) ValidateCallback(ctx context.Context, req authtypes.AuthenticationInput, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	if msg, ok := queryError(req.Query); ok {
		return authtypes.AuthenticationResponse{Error: msg}, nil
	}
	code, ok := get(req.Query, "code")
	if !ok || code == nil {
		code, ok = get(req.Body, "code")
	}
	if !truthy(code, ok) {
		return authtypes.AuthenticationResponse{Error: "No code provided"}, nil
	}
	stateKey, _ := get(req.Query, "state")
	key, _ := stateKey.(string)
	var state *dal.Record
	if key != "" {
		var err error
		if state, err = svc.GetState(ctx, key); err != nil {
			return authtypes.AuthenticationResponse{}, err
		}
	}
	if state == nil {
		return authtypes.AuthenticationResponse{Error: "No state provided, or session expired"}, nil
	}
	callback, cok := state.Get("callback_url")
	params := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s&redirect_uri=%s",
		jsString(s.clientID, true), jsString(s.clientSecret, true), jsString(code, true), jsString(callback, cok))

	res, err := s.exchangeAndUpsert(ctx, params, svc)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	return res, nil
}

func (s *GithubAuthService) exchangeAndUpsert(ctx context.Context, params string, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whatwgURL("https://github.com/login/oauth/access_token?"+params), http.NoBody)
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	req.Header.Set("Accept", "application/json")
	r, err := fetch(req)
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	if !r.ok {
		return authtypes.AuthenticationResponse{}, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Could not exchange token, %d, %s", r.status, r.statusText))
	}
	response, err := r.json()
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}

	accessToken, aok, err := readProperty(response, true, "access_token")
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	refreshToken, rok, _ := readProperty(response, true, "refresh_token")
	expiresIn, eok, _ := readProperty(response, true, "expires_in")
	accessExpires, err := isoDateAfter(jsNumber(expiresIn, eok))
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	refreshExpiresIn, reok, _ := readProperty(response, true, "refresh_token_expires_in")
	refreshExpires, err := isoDateAfter(jsNumber(refreshExpiresIn, reok))
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	providerMetadata := dal.NewRecord()
	if aok {
		providerMetadata.Set("access_token", accessToken)
	}
	if rok {
		providerMetadata.Set("refresh_token", refreshToken)
	}
	providerMetadata.Set("access_token_expires_at", accessExpires)
	providerMetadata.Set("refresh_token_expires_at", refreshExpires)

	// upsert_'s { success: false, error } is reduced to { success,
	// authIdentity } by validateCallback.
	identity, success, err := s.upsert(ctx, providerMetadata, svc)
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	return authtypes.AuthenticationResponse{Success: success, AuthIdentity: identity}, nil
}

// upsert mirrors upsert_ (errors it returns as { success: false, error }
// are dropped by the caller, thrown ones are returned).
func (s *GithubAuthService) upsert(ctx context.Context, providerMetadata *dal.Record, svc authtypes.AuthIdentityProviderService) (*dal.Record, bool, error) {
	token, ok := providerMetadata.Get("access_token")
	if !truthy(token, ok) {
		return nil, false, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+jsString(token, true))
	r, err := fetch(req)
	if err != nil {
		return nil, false, err
	}
	user, err := r.json()
	if err != nil {
		return nil, false, err
	}
	id, idok, err := readProperty(user, true, "id")
	if err != nil {
		return nil, false, err
	}
	if !idok || id == nil {
		return nil, false, fmt.Errorf("Cannot read properties of %s (reading 'toString')", jsString(id, idok))
	}
	entityID := jsString(id, true)
	userMetadata := dal.NewRecord()
	for _, f := range [][2]string{{"profile_url", "url"}, {"avatar", "avatar_url"}, {"email", "email"}, {"name", "name"}, {"company", "company"}} {
		if v, ok := get(user, f[1]); ok {
			userMetadata.Set(f[0], v)
		}
	}
	tfa, ok := get(user, "two_factor_authentication")
	if !ok || tfa == nil {
		tfa = false
	}
	userMetadata.Set("two_factor_authentication", tfa)

	data := dal.NewRecord()
	data.Set("provider_metadata", providerMetadata)
	data.Set("user_metadata", userMetadata)
	identity, err := svc.Update(ctx, entityID, data)
	if err != nil {
		var me *common.MedusaError
		if errors.As(err, &me) && me.Type == common.MedusaErrorTypes.NOT_FOUND {
			created, cerr := svc.Create(ctx, authtypes.CreateAuthIdentityInput{EntityID: entityID, UserMetadata: userMetadata, ProviderMetadata: providerMetadata})
			if cerr != nil {
				return nil, false, cerr
			}
			return created, true, nil
		}
		return nil, false, nil
	}
	return identity, true, nil
}

func (s *GithubAuthService) getRedirect(clientID, callbackURL, stateKey string) authtypes.AuthenticationResponse {
	location := searchParamsURL("https://github.com/login/oauth/authorize", [][2]string{
		{"redirect_uri", callbackURL}, {"client_id", clientID}, {"response_type", "code"}, {"state", stateKey},
	})
	return authtypes.AuthenticationResponse{Success: true, Location: location}
}
