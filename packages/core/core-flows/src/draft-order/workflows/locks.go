package workflows

import (
	lockingsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/locking/steps"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// lockKey is the string lock key of an id (none when it is not a string).
func lockKey(v any) string {
	s, _ := v.(string)
	return s
}

// acquireLock mirrors acquireLockStep({ key, timeout: 2, ttl: 10 }).
func acquireLock(e *composer.Execution, key any) error {
	_, err := lockingsteps.AcquireLockStep.Run(e, lockingsteps.AcquireLockStepInput{Key: lockingsteps.Key(lockKey(key)), Timeout: 2, TTL: 10})
	return err
}

// releaseLock mirrors releaseLockStep({ key }).
func releaseLock(e *composer.Execution, key any) error {
	_, err := lockingsteps.ReleaseLockStep.Run(e, lockingsteps.ReleaseLockStepInput{Key: lockingsteps.Key(lockKey(key))})
	return err
}
