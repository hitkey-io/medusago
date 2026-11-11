// Package services mirrors @medusajs/auth-google/src/services.
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
	// Identifier is GoogleAuthService.identifier.
	Identifier = "google"
	// DisplayName is GoogleAuthService.DISPLAY_NAME.
	DisplayName = "Google Authentication"

	googleJwksURI = "https://www.googleapis.com/oauth2/v3/certs"
)

var googleIssuers = []string{"https://accounts.google.com", "accounts.google.com"}

var _ authtypes.AuthProvider = (*GoogleAuthService)(nil)

// GoogleAuthService mirrors GoogleAuthService.
type GoogleAuthService struct {
	clientID     any
	clientSecret any
	callbackURL  any
	logger       *slog.Logger
	jwks         *jwksClient
}

// ValidateOptions mirrors validateOptions.
func ValidateOptions(options map[string]any) error {
	if !truthy(options["clientId"], options["clientId"] != nil) {
		return errors.New("Google clientId is required")
	}
	if !truthy(options["clientSecret"], options["clientSecret"] != nil) {
		return errors.New("Google clientSecret is required")
	}
	if !truthy(options["callbackUrl"], options["callbackUrl"] != nil) {
		return errors.New("Google callbackUrl is required")
	}
	return nil
}

// NewGoogleAuthService mirrors the constructor (options clientId,
// clientSecret, callbackUrl).
func NewGoogleAuthService(options map[string]any, logger *slog.Logger) (*GoogleAuthService, error) {
	if err := ValidateOptions(options); err != nil {
		return nil, err
	}
	return &GoogleAuthService{
		clientID: options["clientId"], clientSecret: options["clientSecret"], callbackURL: options["callbackUrl"],
		logger: logger, jwks: newJwksClient(googleJwksURI),
	}, nil
}

func (s *GoogleAuthService) Identifier() string  { return Identifier }
func (s *GoogleAuthService) DisplayName() string { return DisplayName }
func (s *GoogleAuthService) IsRedirect() bool    { return true }

// Register mirrors register.
func (s *GoogleAuthService) Register(context.Context, authtypes.AuthenticationInput, authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	return authtypes.AuthenticationResponse{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Google does not support registration. Use method `authenticate` instead.")
}

// Update mirrors AbstractAuthModuleProvider.update.
func (s *GoogleAuthService) Update(context.Context, *dal.Record, authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	return authtypes.AuthenticationResponse{}, fmt.Errorf("Method 'update' not implemented for provider %s", Identifier)
}

func queryError(query *dal.Record) (string, bool) {
	e, ok := get(query, "error")
	if !truthy(e, ok) {
		return "", false
	}
	desc, dok := get(query, "error_description")
	uri, uok := get(query, "error_uri")
	return fmt.Sprintf("%s, read more at: %s", jsString(desc, dok), jsString(uri, uok)), true
}

// Authenticate mirrors authenticate.
func (s *GoogleAuthService) Authenticate(ctx context.Context, req authtypes.AuthenticationInput, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
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
	location := searchParamsURL("https://accounts.google.com/o/oauth2/v2/auth", [][2]string{
		{"redirect_uri", jsString(callback, true)}, {"client_id", jsString(s.clientID, true)}, {"response_type", "code"},
		{"scope", "email profile openid"}, {"state", stateKey},
	})
	return authtypes.AuthenticationResponse{Success: true, Location: location}, nil
}

// ValidateCallback mirrors validateCallback: the code is exchanged for an
// id_token, verified against Google's signing keys.
func (s *GoogleAuthService) ValidateCallback(ctx context.Context, req authtypes.AuthenticationInput, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
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
	params := fmt.Sprintf("client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&grant_type=authorization_code",
		jsString(s.clientID, true), jsString(s.clientSecret, true), jsString(code, true), jsString(callback, cok))
	res, err := s.exchangeAndVerify(ctx, params, svc)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	return res, nil
}

func (s *GoogleAuthService) exchangeAndVerify(ctx context.Context, params string, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whatwgURL("https://oauth2.googleapis.com/token?"+params), http.NoBody)
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
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
	idToken, ok, err := readProperty(response, true, "id_token")
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	// verify_'s { success: false, error } is reduced to { success,
	// authIdentity } by validateCallback.
	identity, success, err := s.verify(ctx, idToken, ok, svc)
	if err != nil {
		return authtypes.AuthenticationResponse{}, err
	}
	return authtypes.AuthenticationResponse{Success: success, AuthIdentity: identity}, nil
}

// verify mirrors verify_.
func (s *GoogleAuthService) verify(ctx context.Context, idToken any, defined bool, svc authtypes.AuthIdentityProviderService) (*dal.Record, bool, error) {
	if !truthy(idToken, defined) {
		return nil, false, nil
	}
	payload, err := verifyIDToken(ctx, idToken, defined, s.jwks, jsString(s.clientID, true), googleIssuers)
	if err != nil {
		return nil, false, common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "Could not verify Google id_token: "+err.Error())
	}
	if v, ok := payload.Get("email_verified"); !truthy(v, ok) {
		return nil, false, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Email not verified, cannot proceed with authentication")
	}
	sub, ok := payload.Get("sub")
	if !truthy(sub, ok) {
		return nil, false, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "id_token is missing 'sub' claim")
	}
	entityID := jsString(sub, true)
	userMetadata := dal.NewRecord()
	for _, k := range []string{"name", "email", "picture", "given_name", "family_name"} {
		if v, ok := payload.Get(k); ok {
			userMetadata.Set(k, v)
		}
	}
	identity, err := svc.Retrieve(ctx, entityID)
	if err != nil {
		var me *common.MedusaError
		if errors.As(err, &me) && me.Type == common.MedusaErrorTypes.NOT_FOUND {
			created, cerr := svc.Create(ctx, authtypes.CreateAuthIdentityInput{EntityID: entityID, UserMetadata: userMetadata})
			if cerr != nil {
				return nil, false, cerr
			}
			return created, true, nil
		}
		return nil, false, nil
	}
	return identity, true, nil
}
