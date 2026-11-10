// Package workflows mirrors @medusajs/core-flows/src/auth/workflows.
package workflows

import (
	"strconv"

	"github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/auth"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ResetPasswordTokenTTLSeconds mirrors RESET_PASSWORD_TOKEN_TTL_SECONDS.
const ResetPasswordTokenTTLSeconds = 15 * 60

// GenerateResetPasswordTokenInput mirrors GenerateResetPasswordTokenWorkflowInput
// (the RBAC grant checks are Enterprise and not ported).
type GenerateResetPasswordTokenInput struct {
	EntityID  string
	ActorType string
	Provider  string
	Secret    string
	Metadata  *dal.Record
}

// GenerateResetPasswordTokenWorkflow mirrors generateResetPasswordTokenWorkflow.
var GenerateResetPasswordTokenWorkflow = composer.CreateWorkflow("generate-reset-password-token",
	func(e *composer.Execution, in GenerateResetPasswordTokenInput) (string, error) {
		reset, err := steps.CreatePasswordResetTokenStep.Run(e, steps.CreatePasswordResetTokenStepInput{EntityID: in.EntityID, Provider: in.Provider, TTLSeconds: ResetPasswordTokenTTLSeconds})
		if err != nil {
			return "", err
		}
		claims := dal.NewRecord()
		claims.Set("entity_id", in.EntityID)
		claims.Set("provider", in.Provider)
		claims.Set("actor_type", in.ActorType)
		claims.Set("purpose", "reset")
		token, err := auth.GenerateJwtTokenWithID(claims, in.Secret, strconv.Itoa(ResetPasswordTokenTTLSeconds), reset.JTI)
		if err != nil {
			return "", err
		}
		metadata := in.Metadata
		if metadata == nil {
			metadata = dal.NewRecord()
		}
		payload := dal.NewRecord()
		payload.Set("entity_id", in.EntityID)
		payload.Set("actor_type", in.ActorType)
		payload.Set("token", token)
		payload.Set("metadata", metadata)
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "auth.password_reset", Data: payload}); err != nil {
			return "", err
		}
		return token, nil
	})
