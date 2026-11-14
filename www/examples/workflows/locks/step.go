package locks

import (
	"context"
	"errors"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// region: step
// lockedChargeCustomerStep acquires and releases the lock itself, with the
// Locking Module's service.
var lockedChargeCustomerStep = composer.CreateStep("locked-charge-customer",
	func(ctx composer.StepExecutionContext, input WorkflowInput) (composer.StepResponse[string], error) {
		locking, ok := ctx.Container.Resolve(utilsmodulessdk.LOCKING).(lockingtypes.ILockingModule)
		if !ok {
			return composer.StepResponse[string]{}, errors.New("the Locking Module isn't registered")
		}

		err := locking.Acquire(ctx.Context, []string{input.OrderID}, lockingtypes.AcquireArgs{
			// Lock will auto-expire after 10 seconds
			Expire: 10,
		})
		if err != nil {
			return composer.StepResponse[string]{}, err
		}

		// TODO charge customer

		if _, err := locking.Release(ctx.Context, []string{input.OrderID}, lockingtypes.ReleaseArgs{}); err != nil {
			return composer.StepResponse[string]{}, err
		}
		return composer.NewStepResponse("charged"), nil
	}, nil)

// endregion: step

// region: execute
// chargeWithExecute holds the lock while a function runs. The lock is
// released when the function returns.
var chargeWithExecute = composer.CreateStep("charge-with-execute",
	func(ctx composer.StepExecutionContext, input WorkflowInput) (composer.StepResponse[string], error) {
		locking := ctx.Container.Resolve(utilsmodulessdk.LOCKING).(lockingtypes.ILockingModule)

		_, err := locking.Execute(ctx.Context, []string{input.OrderID}, func(ctx context.Context) (any, error) {
			// TODO charge customer
			return nil, nil
		}, lockingtypes.ExecuteArgs{Timeout: lockingtypes.Seconds(5)})
		if err != nil {
			return composer.StepResponse[string]{}, err
		}
		return composer.NewStepResponse("charged"), nil
	}, nil)

// endregion: execute
