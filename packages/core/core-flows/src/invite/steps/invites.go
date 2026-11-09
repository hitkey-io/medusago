// Package steps mirrors @medusajs/core-flows/src/invite/steps (the RBAC
// role steps are Enterprise features and are not ported).
package steps

import (
	"context"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type inviteService interface {
	CreateInvites(ctx context.Context, data []*dal.Record) ([]*dal.Record, error)
	RefreshInviteTokens(ctx context.Context, ids []any) ([]*dal.Record, error)
	ValidateInviteToken(ctx context.Context, token string) (*dal.Record, error)
	Internal(className string) *utilsmodulessdk.InternalService
}

func service(c composer.Container) inviteService {
	return c.Resolve(utilsmodulessdk.USER).(inviteService)
}

// CreateInviteStep mirrors createInviteStep.
var CreateInviteStep = composer.CreateStep("create-invite-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		created, err := service(ctx.Container).CreateInvites(ctx.Context, input)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(created, any(commonsteps.RecordIDs(created))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := service(ctx.Container).Internal("Invite").Delete(ctx.Context, ids)
		return err
	})

// DeleteInvitesStep mirrors deleteInvitesStep.
var DeleteInvitesStep = commonsteps.SoftDeleteRecordsStep("delete-invites-step", utilsmodulessdk.USER, "Invite")

// RefreshInviteTokensStep mirrors refreshInviteTokensStep.
var RefreshInviteTokensStep = composer.CreateStep("refresh-invite-tokens-step",
	func(ctx composer.StepExecutionContext, ids []any) (composer.StepResponse[[]*dal.Record], error) {
		invites, err := service(ctx.Container).RefreshInviteTokens(ctx.Context, ids)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(invites), nil
	}, nil)

// ValidateTokenStep mirrors validateTokenStep.
var ValidateTokenStep = composer.CreateStep("validate-invite-token-step",
	func(ctx composer.StepExecutionContext, token string) (composer.StepResponse[*dal.Record], error) {
		invite, err := service(ctx.Container).ValidateInviteToken(ctx.Context, token)
		if err != nil {
			return composer.StepResponse[*dal.Record]{}, err
		}
		return composer.NewStepResponse(invite), nil
	}, nil)
