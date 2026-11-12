// Package steps mirrors @medusajs/core-flows/src/locking/steps.
package steps

import (
	"errors"
	"time"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// LockingModule resolves the locking module (Modules.LOCKING).
func LockingModule(c composer.Container) (lockingtypes.ILockingModule, error) {
	if c != nil {
		if m, ok := c.Resolve(utilsmodulessdk.LOCKING).(lockingtypes.ILockingModule); ok {
			return m, nil
		}
	}
	return nil, errors.New("Could not resolve 'locking'.")
}

// AcquireLockStepInput mirrors AcquireLockStepInput.
type AcquireLockStepInput struct {
	// Key is the key (or keys) to lock.
	Key []string
	// Timeout (seconds) to keep retrying the acquisition (default 0).
	Timeout float64
	// RetryInterval (seconds) between attempts (nil: 0.3).
	RetryInterval *float64
	// TTL (seconds) of the lock (0: no expiration).
	TTL float64
	// OwnerID of the lock.
	OwnerID *string
	// Provider of the locking module (default provider when empty).
	Provider string
	// ExecuteOnSubWorkflow also locks when the workflow runs as a step.
	ExecuteOnSubWorkflow bool
}

// Key builds the Key of the lock inputs from one key (none when empty,
// like an undefined key upstream).
func Key(key string) []string {
	if key == "" {
		return nil
	}
	return []string{key}
}

// AcquireLockStepID mirrors acquireLockStepId.
const AcquireLockStepID = "acquire-lock-step"

// lockCompensation is the compensation input { keys, ownerId, provider }.
func lockCompensation(keys []string, ownerID *string, provider string) *dal.Record {
	r := dal.NewRecord()
	ks := make([]any, len(keys))
	for i, k := range keys {
		ks[i] = k
	}
	r.Set("keys", ks)
	if ownerID != nil {
		r.Set("ownerId", *ownerID)
	}
	if provider != "" {
		r.Set("provider", provider)
	}
	return r
}

// AcquireLockStep mirrors acquireLockStep: the lock is retried every
// retryInterval until timeout; skipped in a workflow run as a step unless
// executeOnSubWorkflow. The compensation releases the keys.
var AcquireLockStep = composer.CreateStep(AcquireLockStepID,
	func(ctx composer.StepExecutionContext, data AcquireLockStepInput) (composer.StepResponse[any], error) {
		keys := data.Key
		if len(keys) == 0 {
			return composer.NewStepResponse[any](nil), nil
		}
		if ctx.IsSubWorkflow() && !data.ExecuteOnSubWorkflow {
			return composer.SkipStep[any](), nil
		}
		locking, err := LockingModule(ctx.Container)
		if err != nil {
			return composer.StepResponse[any]{}, err
		}
		retryInterval := 0.3
		if data.RetryInterval != nil {
			retryInterval = *data.RetryInterval
		}
		tryUntil := time.Now().Add(time.Duration(data.Timeout * float64(time.Second)))
		for {
			err := locking.Acquire(ctx.Context, keys, lockingtypes.AcquireArgs{Expire: data.TTL, OwnerID: data.OwnerID, Provider: data.Provider})
			if err == nil {
				break
			}
			if !time.Now().Before(tryUntil) {
				return composer.StepResponse[any]{}, err
			}
			time.Sleep(time.Duration(retryInterval * float64(time.Second)))
		}
		return composer.NewStepResponse[any](nil, lockCompensation(keys, data.OwnerID, data.Provider)), nil
	},
	func(ctx composer.StepExecutionContext, data any) error {
		r, ok := data.(*dal.Record)
		if !ok || r == nil {
			return nil
		}
		rawKeys, _ := r.Get("keys")
		list, _ := rawKeys.([]any)
		if len(list) == 0 {
			return nil
		}
		keys := make([]string, 0, len(list))
		for _, k := range list {
			if s, ok := k.(string); ok {
				keys = append(keys, s)
			}
		}
		args := lockingtypes.ReleaseArgs{}
		if v, ok := r.Get("ownerId"); ok {
			if s, ok := v.(string); ok {
				args.OwnerID = &s
			}
		}
		if v, ok := r.Get("provider"); ok {
			args.Provider, _ = v.(string)
		}
		locking, err := LockingModule(ctx.Container)
		if err != nil {
			return err
		}
		if _, err = locking.Release(ctx.Context, keys, args); err != nil {
			return err
		}
		ctx.ReturnStepResponse()
		return nil
	})
