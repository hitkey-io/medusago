// Package services mirrors @medusajs/auth-emailpass/src/services.
package services

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"

	authtypes "github.com/hitkey-io/medusago/packages/core/types/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Identifier mirrors EmailPassAuthService.identifier.
const Identifier = "emailpass"

// DisplayName mirrors EmailPassAuthService.DISPLAY_NAME.
const DisplayName = "Email/Password Authentication"

// EmailPassAuthService mirrors EmailPassAuthService.
type EmailPassAuthService struct {
	hashConfig ScryptParams
	logger     *slog.Logger
}

// NewEmailPassAuthService builds the provider from its options
// (hashConfig: { logN, r, p }).
func NewEmailPassAuthService(options map[string]any, logger *slog.Logger) *EmailPassAuthService {
	hc := ScryptParams{LogN: 15, R: 8, P: 1}
	if m, ok := options["hashConfig"].(map[string]any); ok {
		if v, ok := m["logN"].(float64); ok {
			hc.LogN = uint8(v)
		}
		if v, ok := m["r"].(float64); ok {
			hc.R = uint32(v)
		}
		if v, ok := m["p"].(float64); ok {
			hc.P = uint32(v)
		}
	}
	return &EmailPassAuthService{hashConfig: hc, logger: logger}
}

func (s *EmailPassAuthService) Identifier() string  { return Identifier }
func (s *EmailPassAuthService) DisplayName() string { return DisplayName }
func (s *EmailPassAuthService) IsRedirect() bool    { return false }

func (s *EmailPassAuthService) hashPassword(password string) (string, error) {
	h, err := ScryptKdf(password, s.hashConfig)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(h), nil
}

func isNotFound(err error) bool {
	var me *common.MedusaError
	return errors.As(err, &me) && me.Type == common.MedusaErrorTypes.NOT_FOUND
}

// Update mirrors update.
func (s *EmailPassAuthService) Update(ctx context.Context, data *dal.Record, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	var entityID, password string
	if data != nil {
		v, _ := data.Get("entity_id")
		entityID, _ = v.(string)
		p, _ := data.Get("password")
		password, _ = p.(string)
	}
	if entityID == "" {
		return authtypes.AuthenticationResponse{Error: "Cannot update " + Identifier + " provider identity without entity_id"}, nil
	}
	if password == "" {
		return authtypes.AuthenticationResponse{Success: true}, nil
	}
	hash, err := s.hashPassword(password)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	identity, err := svc.Retrieve(ctx, entityID)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	meta := dal.NewRecord()
	if pi := providerIdentity(identity); pi != nil {
		if pm, ok := pi.Get("provider_metadata"); ok {
			if r, ok := pm.(*dal.Record); ok {
				for _, k := range r.Keys() {
					v, _ := r.Get(k)
					meta.Set(k, v)
				}
			}
		}
	}
	meta.Set("password", hash)
	upd := dal.NewRecord()
	upd.Set("provider_metadata", meta)
	identity, err = svc.Update(ctx, entityID, upd)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	return authtypes.AuthenticationResponse{Success: true, AuthIdentity: sanitize(identity)}, nil
}

// Authenticate mirrors authenticate.
func (s *EmailPassAuthService) Authenticate(ctx context.Context, in authtypes.AuthenticationInput, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	password, ok := in.BodyString("password")
	if !ok {
		return authtypes.AuthenticationResponse{Error: "Password should be a string"}, nil
	}
	email, ok := in.BodyString("email")
	if !ok {
		return authtypes.AuthenticationResponse{Error: "Email should be a string"}, nil
	}
	identity, err := svc.Retrieve(ctx, email)
	if err != nil {
		if isNotFound(err) {
			return authtypes.AuthenticationResponse{Error: "Invalid email or password"}, nil
		}
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	if s.isPasswordMatch(password, identity) {
		return authtypes.AuthenticationResponse{Success: true, AuthIdentity: sanitize(identity)}, nil
	}
	return authtypes.AuthenticationResponse{Error: "Invalid email or password"}, nil
}

// Register mirrors register.
func (s *EmailPassAuthService) Register(ctx context.Context, in authtypes.AuthenticationInput, svc authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	password, ok := in.BodyString("password")
	if !ok {
		return authtypes.AuthenticationResponse{Error: "Password should be a string"}, nil
	}
	email, ok := in.BodyString("email")
	if !ok {
		return authtypes.AuthenticationResponse{Error: "Email should be a string"}, nil
	}
	identity, err := svc.Retrieve(ctx, email)
	if err == nil {
		if !isPresent(identity, "app_metadata") && s.isPasswordMatch(password, identity) {
			return authtypes.AuthenticationResponse{Success: true, AuthIdentity: sanitize(identity)}, nil
		}
		return authtypes.AuthenticationResponse{Error: "Identity with email already exists"}, nil
	}
	if !isNotFound(err) {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	hash, err := s.hashPassword(password)
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	meta := dal.NewRecord()
	meta.Set("password", hash)
	created, err := svc.Create(ctx, authtypes.CreateAuthIdentityInput{EntityID: email, ProviderMetadata: meta})
	if err != nil {
		return authtypes.AuthenticationResponse{Error: err.Error()}, nil
	}
	return authtypes.AuthenticationResponse{Success: true, AuthIdentity: sanitize(created)}, nil
}

// ValidateCallback mirrors the AbstractAuthModuleProvider default.
func (s *EmailPassAuthService) ValidateCallback(context.Context, authtypes.AuthenticationInput, authtypes.AuthIdentityProviderService) (authtypes.AuthenticationResponse, error) {
	return authtypes.AuthenticationResponse{}, errors.New("Callback authentication not implemented for provider " + Identifier)
}

// isPresent mirrors isPresent for an object value: defined, non-null and
// not empty.
func isPresent(r *dal.Record, key string) bool {
	v, ok := r.Get(key)
	if !ok || v == nil {
		return false
	}
	if rec, ok := v.(*dal.Record); ok {
		return rec.Len() > 0
	}
	if s, ok := v.(string); ok {
		return s != ""
	}
	return true
}

func providerIdentity(identity *dal.Record) *dal.Record {
	v, _ := identity.Get("provider_identities")
	var list []*dal.Record
	switch t := v.(type) {
	case []*dal.Record:
		list = t
	case []any:
		for _, x := range t {
			if r, ok := x.(*dal.Record); ok {
				list = append(list, r)
			}
		}
	}
	for _, pi := range list {
		if p, _ := pi.Get("provider"); p == Identifier {
			return pi
		}
	}
	return nil
}

func (s *EmailPassAuthService) isPasswordMatch(password string, identity *dal.Record) bool {
	pi := providerIdentity(identity)
	if pi == nil {
		return false
	}
	pm, _ := pi.Get("provider_metadata")
	meta, ok := pm.(*dal.Record)
	if !ok {
		return false
	}
	h, _ := meta.Get("password")
	hash, ok := h.(string)
	if !ok {
		return false
	}
	key, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false
	}
	match, err := ScryptVerify(key, password)
	return err == nil && match
}

// sanitize mirrors sanitizeAuthIdentity_: a JSON copy without the password.
func sanitize(identity *dal.Record) *dal.Record {
	raw, _ := identity.MarshalJSON()
	v, _ := dal.DecodeOrderedJSON(raw)
	cp, ok := v.(*dal.Record)
	if !ok {
		return identity
	}
	if pi := providerIdentity(cp); pi != nil {
		if pm, ok := pi.Get("provider_metadata"); ok {
			if r, ok := pm.(*dal.Record); ok {
				r.Delete("password")
			}
		}
	}
	return cp
}
