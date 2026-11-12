package steps

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type usageService interface {
	RegisterUsage(ctx context.Context, computedActions []*dal.Record, registrationContext *dal.Record) error
	RevertUsage(ctx context.Context, computedActions []*dal.Record, registrationContext *dal.Record) error
}

// RegisterUsageStepInput mirrors RegisterUsageStepInput.
type RegisterUsageStepInput struct {
	ComputedActions     []*dal.Record `json:"computedActions"`
	RegistrationContext *dal.Record   `json:"registrationContext"`
}

// RegisterUsageStep mirrors registerUsageStep.
var RegisterUsageStep = composer.CreateStep("register-usage",
	func(ctx composer.StepExecutionContext, in RegisterUsageStepInput) (composer.StepResponse[any], error) {
		if len(in.ComputedActions) == 0 {
			return composer.NewNullStepResponse[any](any(RegisterUsageStepInput{ComputedActions: []*dal.Record{}, RegistrationContext: in.RegistrationContext})), nil
		}
		svc := ctx.Container.Resolve(utilsmodulessdk.PROMOTION).(usageService)
		if err := svc.RegisterUsage(ctx.Context, in.ComputedActions, in.RegistrationContext); err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewNullStepResponse[any](any(in)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		in, ok := data.(RegisterUsageStepInput)
		if !ok || len(in.ComputedActions) == 0 {
			return nil
		}
		svc := ctx.Container.Resolve(utilsmodulessdk.PROMOTION).(usageService)
		return svc.RevertUsage(ctx.Context, in.ComputedActions, in.RegistrationContext)
	})
