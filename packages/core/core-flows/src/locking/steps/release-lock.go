package steps

import (
	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// ReleaseLockStepInput mirrors ReleaseLockStepInput.
type ReleaseLockStepInput struct {
	// Key is the key (or keys) to release.
	Key []string
	// OwnerID of the lock.
	OwnerID *string
	// Provider of the locking module (default provider when empty).
	Provider string
	// ExecuteOnSubWorkflow also releases when the workflow runs as a step.
	ExecuteOnSubWorkflow bool
}

// ReleaseLockStepID mirrors releaseLockStepId.
const ReleaseLockStepID = "release-lock-step"

// ReleaseLockStep mirrors releaseLockStep: it returns whether the keys were
// released (true when there is no key); skipped in a workflow run as a step
// unless executeOnSubWorkflow.
var ReleaseLockStep = composer.CreateStep(ReleaseLockStepID,
	func(ctx composer.StepExecutionContext, data ReleaseLockStepInput) (composer.StepResponse[any], error) {
		if len(data.Key) == 0 {
			return composer.NewStepResponse[any](true), nil
		}
		if ctx.IsSubWorkflow() && !data.ExecuteOnSubWorkflow {
			return composer.SkipStep[any](), nil
		}
		locking, err := LockingModule(ctx.Container)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		released, err := locking.Release(ctx.Context, data.Key, lockingtypes.ReleaseArgs{OwnerID: data.OwnerID, Provider: data.Provider})
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		return composer.NewStepResponse[any](released), nil
	}, nil)
