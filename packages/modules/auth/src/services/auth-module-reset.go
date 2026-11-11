package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

func hashVerificationToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func recGet(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

// PasswordResetToken is the createPasswordResetToken result.
type PasswordResetToken struct {
	JTI       string
	ExpiresAt time.Time
}

// CreatePasswordResetToken mirrors createPasswordResetToken: previous tokens
// of the provider identity are deleted, the sha256 of a random uuid (the
// jti) is stored.
func (s *AuthModuleService) CreatePasswordResetToken(ctx context.Context, provider, entityID string, ttlSeconds int) (PasswordResetToken, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEvents(ctx)
	var out PasswordResetToken
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		f := dal.NewRecord()
		f.Set("provider", provider)
		f.Set("entity_id", entityID)
		identities, err := tx.Internal("ProviderIdentity").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if len(identities) == 0 {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, fmt.Sprintf("Provider identity with entity_id %s and provider %s not found", entityID, provider))
		}
		identity := identities[0]
		ef := dal.NewRecord()
		ef.Set("provider_identity_id", recGet(identity, "id"))
		existing, err := tx.Internal("AuthPasswordResetToken").List(ctx, ef, utilsmodulessdk.FindConfig{Select: []string{"id"}})
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			ids := make([]any, len(existing))
			for i, t := range existing {
				ids[i] = recGet(t, "id")
			}
			if _, err := tx.Internal("AuthPasswordResetToken").Delete(ctx, ids); err != nil {
				return err
			}
		}
		if ttlSeconds < 1 {
			return common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Password reset token TTL must be a positive integer")
		}
		jti := randomUUID()
		expires := time.Now().Add(time.Duration(ttlSeconds) * time.Second)
		row := dal.NewRecord()
		row.Set("auth_identity_id", recGet(identity, "auth_identity_id"))
		row.Set("provider_identity_id", recGet(identity, "id"))
		row.Set("entity_id", recGet(identity, "entity_id"))
		row.Set("token_hash", hashVerificationToken(jti))
		row.Set("expires_at", expires)
		if _, err := tx.Internal("AuthPasswordResetToken").Create(ctx, []*dal.Record{row}); err != nil {
			return err
		}
		out = PasswordResetToken{JTI: jti, ExpiresAt: expires}
		return nil
	})
	return out, err
}

// ConsumePasswordResetToken mirrors consumePasswordResetToken: the token
// row must exist, be unexpired and belong to the provider identity; it is
// deleted.
func (s *AuthModuleService) ConsumePasswordResetToken(ctx context.Context, jti, provider, entityID string) (*dal.Record, error) {
	// no @EmitEvents upstream: the auth module writes emit no events
	ctx = dal.WithoutEvents(ctx)
	invalid := func() error { return common.NewMedusaError(common.MedusaErrorTypes.UNAUTHORIZED, "Invalid token") }
	var out *dal.Record
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		f := dal.NewRecord()
		f.Set("token_hash", hashVerificationToken(jti))
		tokens, err := tx.Internal("AuthPasswordResetToken").List(ctx, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if len(tokens) == 0 {
			return invalid()
		}
		token := tokens[0]
		expires, _ := recGet(token, "expires_at").(time.Time)
		if !expires.After(time.Now()) {
			if _, err := tx.Internal("AuthPasswordResetToken").Delete(ctx, []any{recGet(token, "id")}); err != nil {
				return err
			}
			return invalid()
		}
		identity, err := tx.Internal("ProviderIdentity").Retrieve(ctx, recGet(token, "provider_identity_id"), utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}
		if recGet(identity, "provider") != provider || recGet(identity, "entity_id") != entityID {
			return invalid()
		}
		if _, err := tx.Internal("AuthPasswordResetToken").Delete(ctx, []any{recGet(token, "id")}); err != nil {
			return err
		}
		out = dal.NewRecord()
		out.Set("auth_identity_id", recGet(token, "auth_identity_id"))
		out.Set("provider_identity_id", recGet(token, "provider_identity_id"))
		out.Set("entity_id", recGet(token, "entity_id"))
		return nil
	})
	return out, err
}
