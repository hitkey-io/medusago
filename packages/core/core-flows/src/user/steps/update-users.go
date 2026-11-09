// Package steps mirrors @medusajs/core-flows/src/user/steps.
package steps

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// UpdateUsersStepID mirrors updateUsersStepId.
const UpdateUsersStepID = "update-users-step"

type internalProvider interface {
	Internal(className string) *utilsmodulessdk.InternalService
}

func userService(c composer.Container) *utilsmodulessdk.InternalService {
	return c.Resolve(utilsmodulessdk.USER).(internalProvider).Internal("User")
}

// UpdateUsersStep mirrors updateUsersStep.
var UpdateUsersStep = composer.CreateStep(UpdateUsersStepID,
	func(ctx composer.StepExecutionContext, input []*dal.Record) (composer.StepResponse[[]*dal.Record], error) {
		if len(input) == 0 {
			return composer.NewStepResponse([]*dal.Record{}, []*dal.Record{}), nil
		}
		svc := userService(ctx.Container)
		ids := make([]any, len(input))
		for i, u := range input {
			ids[i], _ = u.Get("id")
		}
		f := dal.NewRecord()
		f.Set("id", ids)
		original, err := svc.List(ctx.Context, f, utilsmodulessdk.FindConfig{})
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		res, err := svc.Update(ctx.Context, toAny(input))
		if err != nil {
			return composer.StepResponse[[]*dal.Record]{}, err
		}
		users, _ := res.([]*dal.Record)
		return composer.NewStepResponse(users, original), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		original, _ := data.([]*dal.Record)
		if len(original) == 0 {
			return nil
		}
		restore := make([]*dal.Record, len(original))
		for i, u := range original {
			r := dal.NewRecord()
			for _, k := range []string{"id", "first_name", "last_name", "email", "avatar_url", "metadata"} {
				v, _ := u.Get(k)
				r.Set(k, v)
			}
			restore[i] = r
		}
		_, err := userService(ctx.Container).Update(ctx.Context, toAny(restore))
		return err
	})

func toAny(rs []*dal.Record) []any {
	out := make([]any, len(rs))
	for i, r := range rs {
		out[i] = r
	}
	return out
}
