package services

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	cachetypes "github.com/hitkey-io/medusago/packages/core/types/src/cache"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/modules/auth/src/utils"
)

// MFA mirrors the MFA part of AuthModuleService with the built-in "totp"
// and "recovery_code" providers and the default options: the encryption
// key comes from AUTH_MFA_ENCRYPTION_KEY (defineConfig), challenges live
// 300s with 5 attempts in the cache module (Modules.CACHE),
// disable_policy "session", 10 recovery codes.

const (
	totpIssuer           = "Medusa"
	totpDigits           = 6
	totpPeriod           = 30
	totpWindow           = 1
	challengeTTLSeconds  = 300
	challengeMaxAttempt  = 5
	defaultRecoveryCodes = 10
	verificationTTL      = 900 * time.Second
)

func mfaEncryptionKey() (string, error) {
	key := os.Getenv("AUTH_MFA_ENCRYPTION_KEY")
	if key == "" {
		return "", common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "MFA encryption key is required to use MFA methods")
	}
	return key, nil
}

// getCache mirrors getCache_: the cache module is required for the MFA
// challenges.
func (s *AuthModuleService) getCache() (cachetypes.ICacheService, error) {
	if s.container != nil {
		if c, ok := s.container.Resolve(utilsmodulessdk.CACHE).(cachetypes.ICacheService); ok {
			return c, nil
		}
	}
	return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_ARGUMENT, "Cache module dependency is required when using MFA challenges")
}

// jsDate mirrors new Date(value) for a cached date (a Date in memory, an
// ISO string from Redis).
func jsDate(v any) any {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		if d, ok := common.ParseJSDate(t); ok {
			return d
		}
	}
	return v
}

// retrieveMfaChallenge mirrors retrieveMfaChallenge_: nil when the challenge
// is not cached.
func (s *AuthModuleService) retrieveMfaChallenge(ctx context.Context, id string) (*dal.Record, error) {
	cache, err := s.getCache()
	if err != nil {
		return nil, err
	}
	v, err := cache.Get(ctx, challengeKey(id))
	if err != nil {
		return nil, err
	}
	cached, _ := v.(*dal.Record)
	if cached == nil {
		return nil, nil
	}
	c := cloneRecord(cached)
	c.Set("expires_at", jsDate(recGet(c, "expires_at")))
	if completed := recGet(c, "completed_at"); completed != nil && completed != false && completed != "" {
		c.Set("completed_at", jsDate(completed))
	} else {
		c.Set("completed_at", nil)
	}
	return c, nil
}

// setMfaChallenge mirrors setMfaChallenge_: the ttl defaults to the time
// left before expires_at (rounded up); nothing is cached under 1s.
func (s *AuthModuleService) setMfaChallenge(ctx context.Context, c *dal.Record, ttlSeconds *float64) error {
	var ttl float64
	if ttlSeconds != nil {
		ttl = *ttlSeconds
	} else {
		expires, _ := recGet(c, "expires_at").(time.Time)
		ttl = math.Ceil(float64(expires.UnixMilli()-time.Now().UnixMilli()) / 1000)
	}
	if ttl < 1 {
		return nil
	}
	cache, err := s.getCache()
	if err != nil {
		return err
	}
	return cache.Set(ctx, challengeKey(recGet(c, "id")), cloneRecord(c), &ttl)
}

func cloneRecord(r *dal.Record) *dal.Record {
	out := dal.NewRecord()
	for _, k := range r.Keys() {
		v, _ := r.Get(k)
		out.Set(k, v)
	}
	return out
}

func serializeFactor(f *dal.Record) *dal.Record {
	if f == nil {
		return nil
	}
	out := cloneRecord(f)
	out.Delete("provider_metadata")
	return out
}

func asRecord(v any) *dal.Record {
	switch t := v.(type) {
	case *dal.Record:
		return t
	case []*dal.Record:
		if len(t) > 0 {
			return t[0]
		}
	case []any:
		if len(t) > 0 {
			r, _ := t[0].(*dal.Record)
			return r
		}
	}
	return nil
}

func (s *AuthModuleService) listFactors(ctx context.Context, filters *dal.Record, sel []string) ([]*dal.Record, error) {
	return s.Internal("AuthMfaFactor").List(ctx, filters, utilsmodulessdk.FindConfig{Select: sel})
}

func (s *AuthModuleService) retrieveAuthIdentityExists(ctx context.Context, id any) error {
	_, err := s.Internal("AuthIdentity").Retrieve(ctx, id, utilsmodulessdk.FindConfig{})
	return err
}

func mfaMethodError(method string) error {
	return fmt.Errorf("\nUnable to retrieve the MFA method with id: %s\nPlease make sure that the method is registered in the container and it is configured correctly in your project configuration file.", method)
}

// canVerify mirrors provider.canVerifyForAuthIdentity.
func (s *AuthModuleService) canVerify(ctx context.Context, method string, authIdentityID any) (bool, error) {
	switch method {
	case "totp":
		f := dal.NewRecord()
		f.Set("auth_identity_id", authIdentityID)
		f.Set("provider", "totp")
		f.Set("status", "enabled")
		list, err := s.listFactors(ctx, f, []string{"id"})
		return len(list) > 0, err
	case "recovery_code":
		f := dal.NewRecord()
		f.Set("auth_identity_id", authIdentityID)
		list, err := s.Internal("AuthMfaRecoveryCode").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		return len(list) > 0, err
	}
	return false, mfaMethodError(method)
}

func verifyTotpFactor(factor *dal.Record, code string) (bool, error) {
	key, err := mfaEncryptionKey()
	if err != nil {
		return false, err
	}
	meta, _ := recGet(factor, "provider_metadata").(*dal.Record)
	enc, _ := recGet(meta, "secret").(string)
	secret, err := utils.DecryptSecret(enc, key)
	if err != nil {
		return false, err
	}
	return utils.VerifyTotpCode(secret, code, totpDigits, totpPeriod, totpWindow, time.Now()), nil
}

// verifyMethod mirrors provider.verify.
func (s *AuthModuleService) verifyMethod(ctx context.Context, method string, authIdentityID any, code string) (bool, error) {
	switch method {
	case "totp":
		f := dal.NewRecord()
		f.Set("auth_identity_id", authIdentityID)
		f.Set("provider", "totp")
		f.Set("status", "enabled")
		list, err := s.listFactors(ctx, f, nil)
		if err != nil || len(list) == 0 {
			return false, err
		}
		return verifyTotpFactor(list[0], code)
	case "recovery_code":
		f := dal.NewRecord()
		f.Set("auth_identity_id", authIdentityID)
		codes, err := s.Internal("AuthMfaRecoveryCode").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id", "code_hash"}})
		if err != nil {
			return false, err
		}
		for _, c := range codes {
			hash, _ := recGet(c, "code_hash").(string)
			if utils.VerifyRecoveryCodeHash(hash, code) {
				_, err := s.Internal("AuthMfaRecoveryCode").Delete(ctx, []any{recGet(c, "id")})
				return err == nil, err
			}
		}
		return false, nil
	}
	return false, mfaMethodError(method)
}

// UseAuthMfaRecoveryCode mirrors useAuthMfaRecoveryCode: the recovery code
// of the auth identity is verified and consumed in a transaction.
func (s *AuthModuleService) UseAuthMfaRecoveryCode(ctx context.Context, authIdentityID any, code string) error {
	return s.InTransaction(ctx, func(m *utilsmodulessdk.MedusaService) error {
		tx := *s
		tx.MedusaService = m
		valid, err := tx.verifyMethod(ctx, "recovery_code", authIdentityID, code)
		if err != nil {
			return err
		}
		if !valid {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Recovery code is invalid or already used")
		}
		return nil
	})
}

// MfaStartResponse mirrors AuthMfaStartResponse.
type MfaStartResponse struct {
	Mfa        *dal.Record
	Secret     string
	OtpauthURL string
}

// StartAuthMfa mirrors startAuthMfa.
func (s *AuthModuleService) StartAuthMfa(ctx context.Context, authIdentityID any, provider string, label any, issuer string, metadata *dal.Record) (MfaStartResponse, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	var out MfaStartResponse
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		t := &AuthModuleService{MedusaService: tx, providers: s.providers, identifiers: s.identifiers}
		if err := t.retrieveAuthIdentityExists(ctx, authIdentityID); err != nil {
			return err
		}
		switch provider {
		case "totp":
		case "recovery_code":
			return fmt.Errorf("MFA provider \"%s\" does not support setup", provider)
		default:
			return mfaMethodError(provider)
		}
		if issuer == "" {
			issuer = totpIssuer
		}
		f := dal.NewRecord()
		f.Set("auth_identity_id", authIdentityID)
		f.Set("provider", "totp")
		f.Set("status", []any{"pending", "enabled"})
		existing, err := t.listFactors(ctx, f, []string{"id"})
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "An active TOTP factor already exists for this auth identity")
		}
		key, err := mfaEncryptionKey()
		if err != nil {
			return err
		}
		secret := utils.GenerateTotpSecret()
		enc, err := utils.EncryptSecret(secret, key)
		if err != nil {
			return err
		}
		pm := dal.NewRecord()
		pm.Set("secret", enc)
		pm.Set("issuer", issuer)
		meta := dal.NewRecord()
		if metadata != nil {
			for _, k := range metadata.Keys() {
				v, _ := metadata.Get(k)
				meta.Set(k, v)
			}
		}
		if label != nil {
			meta.Set("label", label)
		}
		row := dal.NewRecord()
		row.Set("auth_identity_id", authIdentityID)
		row.Set("provider", "totp")
		row.Set("status", "pending")
		row.Set("provider_metadata", pm)
		row.Set("metadata", meta)
		created, err := t.Internal("AuthMfaFactor").Create(ctx, []*dal.Record{row})
		if err != nil {
			return err
		}
		account := fmt.Sprint(authIdentityID)
		if l, ok := label.(string); ok {
			account = l
		}
		// serialized before the transaction flushes: the timestamps are not
		// set on the entity yet
		mfa := serializeFactor(created[0])
		for _, k := range []string{"created_at", "updated_at", "deleted_at"} {
			mfa.Delete(k)
		}
		out = MfaStartResponse{Mfa: mfa, Secret: secret, OtpauthURL: utils.GenerateTotpURI(issuer, account, secret, totpDigits, totpPeriod)}
		return nil
	})
	return out, err
}

func notFoundFactor(id any) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`MFA factor with id "%v" was not found`, id))
}

// VerifyAuthMfa mirrors verifyAuthMfa (TOTP setup verification).
func (s *AuthModuleService) VerifyAuthMfa(ctx context.Context, id, authIdentityID any, code string) (*dal.Record, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	var out *dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		factor, err := tx.Internal("AuthMfaFactor").Retrieve(ctx, id, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if authIdentityID != nil && authIdentityID != "" && recGet(factor, "auth_identity_id") != authIdentityID {
			return notFoundFactor(id)
		}
		if p := recGet(factor, "provider"); p != "totp" {
			if p == "recovery_code" {
				return fmt.Errorf("MFA provider \"%v\" does not support setup verification", p)
			}
			return mfaMethodError(fmt.Sprint(p))
		}
		if recGet(factor, "status") == "disabled" {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Disabled MFA factors cannot be verified")
		}
		valid, err := verifyTotpFactor(factor, code)
		if err != nil {
			return err
		}
		if !valid {
			return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Invalid TOTP code")
		}
		if recGet(factor, "status") == "pending" {
			u := dal.NewRecord()
			u.Set("id", recGet(factor, "id"))
			u.Set("status", "enabled")
			res, err := tx.Internal("AuthMfaFactor").Update(ctx, u)
			if err != nil {
				return err
			}
			factor = asRecord(res)
		}
		out = serializeFactor(factor)
		return nil
	})
	return out, err
}

// RetrieveAuthMfa mirrors retrieveAuthMfa({ id, auth_identity_id }).
func (s *AuthModuleService) RetrieveAuthMfa(ctx context.Context, id, authIdentityID any) (*dal.Record, error) {
	f := dal.NewRecord()
	f.Set("id", []any{id})
	f.Set("auth_identity_id", authIdentityID)
	list, err := s.listFactors(ctx, f, nil)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, notFoundFactor(id)
	}
	return serializeFactor(list[0]), nil
}

// ListAuthMfa mirrors listAuthMfa.
func (s *AuthModuleService) ListAuthMfa(ctx context.Context, filters *dal.Record) ([]*dal.Record, error) {
	list, err := s.listFactors(ctx, filters, nil)
	if err != nil {
		return nil, err
	}
	out := make([]*dal.Record, len(list))
	for i, f := range list {
		out[i] = serializeFactor(f)
	}
	return out, nil
}

// DisableAuthMfa mirrors disableAuthMfa (disable_policy "session": no code
// needed).
func (s *AuthModuleService) DisableAuthMfa(ctx context.Context, id any) (*dal.Record, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	if _, err := s.Internal("AuthMfaFactor").Retrieve(ctx, id, utilsmodulessdk.FindConfig{}); err != nil {
		return nil, err
	}
	u := dal.NewRecord()
	u.Set("id", id)
	u.Set("status", "disabled")
	res, err := s.Internal("AuthMfaFactor").Update(ctx, u)
	if err != nil {
		return nil, err
	}
	return serializeFactor(asRecord(res)), nil
}

// GenerateAuthMfaRecoveryCodes mirrors generateAuthMfaRecoveryCodes.
func (s *AuthModuleService) GenerateAuthMfaRecoveryCodes(ctx context.Context, authIdentityID any, count *int) ([]string, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	n := defaultRecoveryCodes
	if count != nil {
		n = *count
	}
	if n < 1 || n > 50 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Recovery code count must be between 1 and 50")
	}
	var codes []string
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		if _, err := tx.Internal("AuthIdentity").Retrieve(ctx, authIdentityID, utilsmodulessdk.FindConfig{}); err != nil {
			return err
		}
		f := dal.NewRecord()
		f.Set("auth_identity_id", authIdentityID)
		existing, err := tx.Internal("AuthMfaRecoveryCode").List(ctx, f, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			ids := make([]any, len(existing))
			for i, c := range existing {
				ids[i] = recGet(c, "id")
			}
			if _, err := tx.Internal("AuthMfaRecoveryCode").Delete(ctx, ids); err != nil {
				return err
			}
		}
		rows := make([]*dal.Record, n)
		codes = make([]string, n)
		for i := range codes {
			codes[i] = utils.GenerateRecoveryCode()
			hash, err := utils.HashRecoveryCode(codes[i])
			if err != nil {
				return err
			}
			r := dal.NewRecord()
			r.Set("auth_identity_id", authIdentityID)
			r.Set("code_hash", hash)
			rows[i] = r
		}
		_, err = tx.Internal("AuthMfaRecoveryCode").Create(ctx, rows)
		return err
	})
	return codes, err
}

// availableMfaChallengeMethods mirrors getAvailableMfaChallengeMethods_.
func (s *AuthModuleService) availableMfaChallengeMethods(ctx context.Context, authIdentityID any) ([]any, error) {
	f := dal.NewRecord()
	f.Set("auth_identity_id", authIdentityID)
	f.Set("status", "enabled")
	factors, err := s.listFactors(ctx, f, []string{"provider"})
	if err != nil {
		return nil, err
	}
	var factorMethods []string
	seen := map[string]bool{}
	for _, fa := range factors {
		m := fmt.Sprint(recGet(fa, "provider"))
		if !seen[m] {
			seen[m] = true
			factorMethods = append(factorMethods, m)
		}
	}
	methods := []any{}
	for _, m := range factorMethods {
		ok, err := s.canVerify(ctx, m, authIdentityID)
		if err != nil {
			return nil, err
		}
		if ok {
			methods = append(methods, m)
		}
	}
	if len(methods) == 0 {
		return methods, nil
	}
	if ok, err := s.canVerify(ctx, "recovery_code", authIdentityID); err != nil {
		return nil, err
	} else if ok {
		methods = append(methods, "recovery_code")
	}
	return methods, nil
}

func challengeKey(id any) string { return fmt.Sprintf("auth:mfa:challenge:%v", id) }

// createAuthMfaChallenge mirrors createAuthMfaChallenge_.
func (s *AuthModuleService) createAuthMfaChallenge(ctx context.Context, authIdentityID any, authProvider any, metadata any) (*dal.Record, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	if err := s.retrieveAuthIdentityExists(ctx, authIdentityID); err != nil {
		return nil, err
	}
	methods, err := s.availableMfaChallengeMethods(ctx, authIdentityID)
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Auth identity does not have any enabled MFA methods")
	}
	c := dal.NewRecord()
	c.Set("id", common.GenerateEntityID("", "authmfachal"))
	c.Set("auth_identity_id", authIdentityID)
	c.Set("auth_provider", authProvider)
	c.Set("methods", methods)
	c.Set("expires_at", time.Now().Add(challengeTTLSeconds*time.Second))
	c.Set("attempts", float64(0))
	c.Set("max_attempts", float64(challengeMaxAttempt))
	c.Set("completed_at", nil)
	c.Set("metadata", metadata)
	ttl := float64(challengeTTLSeconds)
	if err := s.setMfaChallenge(ctx, c, &ttl); err != nil {
		return nil, err
	}
	return c, nil
}

// VerifyAuthMfaChallenge mirrors verifyAuthMfaChallenge.
func (s *AuthModuleService) VerifyAuthMfaChallenge(ctx context.Context, id, method, code string, authIdentityID any) (*dal.Record, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	c, err := s.retrieveMfaChallenge(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`MFA challenge with id "%s" was not found`, id))
	}
	if authIdentityID != nil && authIdentityID != "" && recGet(c, "auth_identity_id") != authIdentityID {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`MFA challenge with id "%s" was not found`, id))
	}
	notAllowed := func(m string) error { return common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, m) }
	if recGet(c, "completed_at") != nil {
		return nil, notAllowed("MFA challenge has already been completed")
	}
	if expires, _ := recGet(c, "expires_at").(time.Time); !expires.After(time.Now()) {
		return nil, notAllowed("MFA challenge has expired")
	}
	attempts, _ := recGet(c, "attempts").(float64)
	maxAttempts, _ := recGet(c, "max_attempts").(float64)
	if attempts >= maxAttempts {
		return nil, notAllowed("MFA challenge has too many failed attempts")
	}
	supported := false
	methods, _ := recGet(c, "methods").([]any)
	for _, m := range methods {
		if m == method {
			supported = true
		}
	}
	if !supported {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf(`MFA challenge does not support method "%s"`, method))
	}
	valid, err := s.verifyMethod(ctx, method, recGet(c, "auth_identity_id"), code)
	if err != nil {
		return nil, err
	}
	if !valid {
		attempts++
		c.Set("attempts", attempts)
		if err := s.setMfaChallenge(ctx, c, nil); err != nil {
			return nil, err
		}
		if attempts >= maxAttempts {
			return nil, notAllowed("MFA challenge has too many failed attempts")
		}
		return nil, notAllowed("Invalid MFA challenge code")
	}
	c.Set("completed_at", time.Now())
	if err := s.setMfaChallenge(ctx, c, nil); err != nil {
		return nil, err
	}
	return c, nil
}

// RequestAuthVerification mirrors requestAuthVerification with the token
// verification provider.
func (s *AuthModuleService) RequestAuthVerification(ctx context.Context, authIdentityID, entityID, entityType, codeProvider any, metadata any) (*dal.Record, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	if codeProvider == nil || codeProvider == "" {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Verification provider is required")
	}
	if codeProvider != "token" {
		return nil, fmt.Errorf("\nUnable to retrieve the verification provider with id: %v\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.", codeProvider)
	}
	svc := s.Internal("AuthVerification")
	f := dal.NewRecord()
	f.Set("auth_identity_id", authIdentityID)
	f.Set("entity_id", entityID)
	f.Set("entity_type", entityType)
	take, skip := 1, 0
	existing, err := svc.List(ctx, f, utilsmodulessdk.FindConfig{Take: &take, Skip: &skip})
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 && recGet(existing[0], "verified_at") != nil {
		return existing[0], nil
	}
	token := utils.GenerateVerificationToken()
	pm := dal.NewRecord()
	pm.Set("token_hash", utils.HashVerificationToken(token))
	requested := time.Now()
	var verification *dal.Record
	if len(existing) > 0 {
		u := dal.NewRecord()
		u.Set("id", recGet(existing[0], "id"))
		u.Set("code_provider", codeProvider)
		u.Set("provider_metadata", pm)
		u.Set("requested_at", requested)
		u.Set("verified_at", nil)
		res, err := svc.Update(ctx, u)
		if err != nil {
			return nil, err
		}
		verification = asRecord(res)
	} else {
		row := dal.NewRecord()
		row.Set("auth_identity_id", authIdentityID)
		row.Set("entity_id", entityID)
		row.Set("entity_type", entityType)
		row.Set("code_provider", codeProvider)
		row.Set("provider_metadata", pm)
		row.Set("requested_at", requested)
		row.Set("metadata", metadata)
		created, err := svc.Create(ctx, []*dal.Record{row})
		if err != nil {
			return nil, err
		}
		verification = created[0]
	}
	// { ...verification }: the spread entity has its generated id last
	out := dal.NewRecord()
	for _, k := range verification.Keys() {
		if k == "id" {
			continue
		}
		v, _ := verification.Get(k)
		out.Set(k, v)
	}
	out.Set("id", recGet(verification, "id"))
	out.Set("code", token)
	out.Set("expires_at", requested.Add(verificationTTL))
	return out, nil
}

// ConfirmAuthVerification mirrors confirmAuthVerification (token provider).
func (s *AuthModuleService) ConfirmAuthVerification(ctx context.Context, code string, codeProvider any) (*dal.Record, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEmit(ctx)
	if codeProvider == nil {
		codeProvider = "token"
	}
	if codeProvider != "token" {
		return nil, fmt.Errorf("\nUnable to retrieve the verification provider with id: %v\nPlease make sure that the provider is registered in the container and it is configured correctly in your project configuration file.", codeProvider)
	}
	if code == "" {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Verification code is required")
	}
	svc := s.Internal("AuthVerification")
	pm := dal.NewRecord()
	pm.Set("token_hash", utils.HashVerificationToken(code))
	f := dal.NewRecord()
	f.Set("provider_metadata", pm)
	list, err := svc.List(ctx, f, utilsmodulessdk.FindConfig{})
	if err != nil {
		return nil, err
	}
	if len(list) == 0 || recGet(list[0], "verified_at") != nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Verification code is invalid or already used")
	}
	v := list[0]
	if recGet(v, "code_provider") != codeProvider {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf(`Verification code does not belong to provider "%v"`, codeProvider))
	}
	requested, _ := recGet(v, "requested_at").(time.Time)
	if !requested.Add(verificationTTL).After(time.Now()) {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Verification code has expired")
	}
	u := dal.NewRecord()
	u.Set("id", recGet(v, "id"))
	u.Set("verified_at", time.Now())
	res, err := svc.Update(ctx, u)
	if err != nil {
		return nil, err
	}
	return asRecord(res), nil
}
