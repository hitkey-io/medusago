package workflows

import (
	"github.com/hitkey-io/medusago/packages/core/core-flows/src/auth/steps"
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	usersteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/user/steps"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
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

// CreateUsersWorkflowInput mirrors CreateUsersWorkflowInputDTO.
type CreateUsersWorkflowInput struct {
	Users []*dal.Record
}

// CreateUsersWorkflow mirrors createUsersWorkflow without the RBAC steps
// (role validation and assignments are Enterprise features).
var CreateUsersWorkflow = composer.CreateWorkflow("create-users-workflow",
	func(e *composer.Execution, input CreateUsersWorkflowInput) ([]*dal.Record, error) {
		created, err := usersteps.CreateUsersStep.Run(e, input.Users)
		if err != nil {
			return nil, err
		}
		if _, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "user.created", Data: idEvents(commonsteps.RecordIDs(created))}); err != nil {
			return nil, err
		}
		return created, nil
	})

// DeleteUsersWorkflowInput mirrors DeleteUserWorkflowInput.
type DeleteUsersWorkflowInput struct {
	IDs []any
}

// DeleteUsersWorkflow mirrors deleteUsersWorkflow without the RBAC steps.
var DeleteUsersWorkflow = composer.CreateWorkflow("delete-user",
	func(e *composer.Execution, input DeleteUsersWorkflowInput) (any, error) {
		if _, err := usersteps.DeleteUsersStep.Run(e, input.IDs); err != nil {
			return nil, err
		}
		keys := dal.NewRecord()
		keys.Set("user_id", input.IDs)
		entry := dal.NewRecord()
		entry.Set(utilsmodulessdk.USER, keys)
		if _, err := commonsteps.RemoveRemoteLinkStep.Run(e, []*dal.Record{entry}); err != nil {
			return nil, err
		}
		_, err := commonsteps.EmitEventStep.Run(e, commonsteps.EmitEventStepInput{EventName: "user.deleted", Data: idEvents(input.IDs)})
		return nil, err
	})

// RemoveUserAccountWorkflow mirrors removeUserAccountWorkflow.
var RemoveUserAccountWorkflow = composer.CreateWorkflow("remove-user-account",
	func(e *composer.Execution, userID string) (string, error) {
		if _, err := DeleteUsersWorkflow.RunAsStep(e, DeleteUsersWorkflowInput{IDs: []any{userID}}); err != nil {
			return "", err
		}
		meta := dal.NewRecord()
		meta.Set("user_id", userID)
		f := dal.NewRecord()
		f.Set("app_metadata", meta)
		res, err := commonsteps.UseQueryGraphStep.Config("query-auth-identity", composer.StepOptions{}).Run(e, commonsteps.UseQueryGraphStepInput{
			GraphInput: query.GraphInput{Entity: "auth_identity", Fields: []string{"id", "app_metadata", "provider_identities.entity_id"}, Filters: f},
		})
		if err != nil {
			return "", err
		}
		rows, _ := res.Data.([]any)
		if len(rows) == 0 {
			return userID, nil
		}
		identity, _ := rows[0].(*dal.Record)
		if identity == nil {
			return userID, nil
		}
		keep := false
		if m, ok := identity.Get("app_metadata"); ok {
			if mr, ok := m.(*dal.Record); ok {
				for _, k := range mr.Keys() {
					if k == "user_id" {
						continue
					}
					if v, _ := mr.Get(k); v != nil {
						keep = true
					}
				}
			}
		}
		id, _ := identity.Get("id")
		if keep {
			_, err = steps.SetAuthAppMetadataStep.Run(e, steps.SetAuthAppMetadataStepInput{AuthIdentityID: id, ActorType: "user", Value: nil})
		} else {
			_, err = steps.DeleteAuthIdentityStep.Run(e, []any{id})
		}
		if err != nil {
			return "", err
		}
		return userID, nil
	})
