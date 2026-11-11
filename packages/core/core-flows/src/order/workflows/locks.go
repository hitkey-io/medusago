package workflows

import (
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// acquireLock mirrors acquireLockStep({ key, timeout, ttl }).
func acquireLock(e *composer.Execution, key string, timeout, ttl float64) error {
	_, err := lockingsteps.AcquireLockStep.Run(e, lockingsteps.AcquireLockStepInput{Key: lockingsteps.Key(key), Timeout: timeout, TTL: ttl})
	return err
}

// releaseLock mirrors releaseLockStep({ key }).
func releaseLock(e *composer.Execution, key string) error {
	_, err := lockingsteps.ReleaseLockStep.Run(e, lockingsteps.ReleaseLockStepInput{Key: lockingsteps.Key(key)})
	return err
}

// lockKey is the string lock key of an id (none when it is not a string).
func lockKey(v any) string {
	s, _ := v.(string)
	return s
}

// withOrderLock wraps a workflow body between acquireLockStep({ key: order_id,
// timeout: 2, ttl: 10 }) as its first step and releaseLockStep({ key:
// order_id }) as its last one (the order edit workflows).
func withOrderLock[I, O any](key func(I) any, body func(*composer.Execution, I) (O, error)) func(*composer.Execution, I) (O, error) {
	return func(e *composer.Execution, in I) (O, error) {
		var zero O
		k := lockKey(key(in))
		if err := acquireLock(e, k, 2, 10); err != nil {
			return zero, err
		}
		out, err := body(e, in)
		if err != nil {
			return zero, err
		}
		if err := releaseLock(e, k); err != nil {
			return zero, err
		}
		return out, nil
	}
}
