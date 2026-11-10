package steps

import (
	"context"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	authservices "github.com/hitkey-io/medusago/packages/modules/auth/src/services"
)

// CreatePasswordResetTokenStepInput mirrors CreatePasswordResetTokenStepInput.
type CreatePasswordResetTokenStepInput struct {
	EntityID   string
	Provider   string
	TTLSeconds int
}

// CreatePasswordResetTokenStepOutput mirrors CreatePasswordResetTokenStepOutput.
type CreatePasswordResetTokenStepOutput struct {
	JTI       string `json:"jti"`
	ExpiresAt string `json:"expiresAt"`
}

type resetTokens interface {
	CreatePasswordResetToken(ctx context.Context, provider, entityID string, ttlSeconds int) (authservices.PasswordResetToken, error)
	ConsumePasswordResetToken(ctx context.Context, jti, provider, entityID string) (*dal.Record, error)
}

func resetTokenService(c composer.Container) resetTokens {
	s, _ := c.Resolve(utilsmodulessdk.AUTH).(resetTokens)
	return s
}

// CreatePasswordResetTokenStep mirrors createPasswordResetTokenStep; the
// compensation consumes the token (best effort).
var CreatePasswordResetTokenStep = composer.CreateStep("create-password-reset-token",
	func(ctx composer.StepExecutionContext, in CreatePasswordResetTokenStepInput) (composer.StepResponse[CreatePasswordResetTokenStepOutput], error) {
		t, err := resetTokenService(ctx.Container).CreatePasswordResetToken(ctx.Context, in.Provider, in.EntityID, in.TTLSeconds)
		if err != nil {
			return composer.StepResponse[CreatePasswordResetTokenStepOutput]{}, err
		}
		comp := dal.NewRecord()
		comp.Set("jti", t.JTI)
		comp.Set("provider", in.Provider)
		comp.Set("entityId", in.EntityID)
		return composer.NewStepResponse(CreatePasswordResetTokenStepOutput{JTI: t.JTI, ExpiresAt: dal.FormatJSDate(t.ExpiresAt.In(time.UTC))}, any(comp)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, _ := data.(*dal.Record)
		if r == nil {
			return nil
		}
		jti, _ := r.Get("jti")
		provider, _ := r.Get("provider")
		entity, _ := r.Get("entityId")
		_, _ = resetTokenService(ctx.Container).ConsumePasswordResetToken(ctx.Context, jti.(string), provider.(string), entity.(string))
		return nil
	})
