// Package workflows mirrors @medusajs/core-flows/src/invite/workflows
// without the RBAC role steps (Enterprise features).
package workflows

import (
	authsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/invite/steps"
	userworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/user/workflows"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

func idEvents(ids []any) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		ev := dal.NewRecord()
		ev.Set("id", id)
		out[i] = ev
	}
	return out
}

// CreateInvitesWorkflowInput mirrors CreateInvitesWorkflowInputDTO.
type CreateInvitesWorkflowInput struct {
	Invites []*dal.Record
}

// CreateInvitesWorkflow mirrors createInvitesWorkflow.
var CreateInvitesWorkflow = composer.CreateWorkflow("create-invite-step",
	func(e *composer.Execution, input CreateInvitesWorkflowInput) ([]*dal.Record, error) {
		created, err := steps.CreateInviteStep.Run(e, input.Invites)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "invite.created", Data: idEvents(commonsteps.RecordIDs(created))}); err != nil {
			return nil, err
		}
		return created, nil
	})

// DeleteInvitesWorkflow mirrors deleteInvitesWorkflow.
var DeleteInvitesWorkflow = composer.CreateWorkflow("delete-invites-workflow",
	func(e *composer.Execution, ids []any) (any, error) {
		if _, err := steps.DeleteInvitesStep.Run(e, ids); err != nil {
			return nil, err
		}
		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "invite.deleted", Data: idEvents(ids)})
		return nil, err
	})

// RefreshInviteTokensWorkflow mirrors refreshInviteTokensWorkflow.
var RefreshInviteTokensWorkflow = composer.CreateWorkflow("refresh-invite-tokens-workflow",
	func(e *composer.Execution, ids []any) ([]*dal.Record, error) {
		invites, err := steps.RefreshInviteTokensStep.Run(e, ids)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "invite.resent", Data: idEvents(commonsteps.RecordIDs(invites))}); err != nil {
			return nil, err
		}
		return invites, nil
	})

// AcceptInviteWorkflowInput mirrors AcceptInviteWorkflowInputDTO.
type AcceptInviteWorkflowInput struct {
	InviteToken    string
	AuthIdentityID any
	User           *dal.Record
}

// AcceptInviteWorkflow mirrors acceptInviteWorkflow.
var AcceptInviteWorkflow = composer.CreateWorkflow("accept-invite-workflow",
	func(e *composer.Execution, input AcceptInviteWorkflowInput) ([]*dal.Record, error) {
		invite, err := steps.ValidateTokenStep.Run(e, input.InviteToken)
		if err != nil {
			return nil, err
		}
		u := dal.NewRecord()
		if input.User != nil {
			for _, k := range input.User.Keys() {
				v, _ := input.User.Get(k)
				u.Set(k, v)
			}
		}
		if v, _ := u.Get("email"); v == nil {
			email, _ := invite.Get("email")
			u.Set("email", email)
		}
		users, err := userworkflows.CreateUsersWorkflow.RunAsStep(e, userworkflows.CreateUsersWorkflowInput{Users: []*dal.Record{u}})
		if err != nil {
			return nil, err
		}
		userID, _ := users[0].Get("id")
		inviteID, _ := invite.Get("id")
		// parallelize(setAuthAppMetadata, deleteInvites, ..., emitEvent): the
		// event step (no I/O) completes first upstream
		ev := dal.NewRecord()
		ev.Set("id", inviteID)
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "invite.accepted", Data: ev}); err != nil {
			return nil, err
		}
		if _, err := authsteps.SetAuthAppMetadataStep.Run(e, authsteps.SetAuthAppMetadataStepInput{AuthIdentityID: input.AuthIdentityID, ActorType: "user", Value: userID}); err != nil {
			return nil, err
		}
		if _, err := steps.DeleteInvitesStep.Run(e, []any{inviteID}); err != nil {
			return nil, err
		}
		return users, nil
	})
