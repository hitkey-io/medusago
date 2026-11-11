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
