package steps

import (
	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// CreateUsersStep mirrors createUsersStep (role assignments belong to the
// RBAC module and are not part of the user input).
var CreateUsersStep = composer.CreateStep("create-users-step",
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		data := make([]*dal.Record, len(input))
		for i, u := range input {
			n := dal.NewRecord()
			for _, k := range u.Keys() {
				if k == "roles" {
					continue
				}
				v, _ := u.Get(k)
				n.Set(k, v)
			}
			data[i] = n
		}
		users, err := userService(ctx.Container).Create(ctx.Context, data)
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		return composer.NewStepResponse(users, any(commonsteps.RecordIDs(users))), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		ids, _ := data.([]any)
		if len(ids) == 0 {
			return nil
		}
		_, err := userService(ctx.Container).Delete(ctx.Context, ids)
		return err
	})

// DeleteUsersStep mirrors deleteUsersStep.
var DeleteUsersStep = commonsteps.SoftDeleteRecordsStep("delete-users-step", utilsmodulessdk.USER, "User")
