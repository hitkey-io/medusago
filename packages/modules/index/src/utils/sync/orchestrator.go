// Package sync mirrors @medusajs/index/src/utils/sync.
package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
)

// Orchestrator mirrors Orchestrator: processes the entities one by one,
// each under a lock of the locking module (owner index-sync-<pid>), so
// several processes do not sync the same entity.
type Orchestrator struct {
	locking      lockingtypes.ILockingModule
	logger       *slog.Logger
	lockingOwner string
	state        string
	// lockDuration in seconds
	lockDuration float64
	entities     []string
}

// NewOrchestrator mirrors new Orchestrator(lockingModule, entities, options).
func NewOrchestrator(locking lockingtypes.ILockingModule, entities []string, lockDuration float64, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{locking: locking, logger: logger, lockingOwner: fmt.Sprintf("index-sync-%d", os.Getpid()), state: "idle", lockDuration: lockDuration, entities: entities}
}

// acquireLock mirrors #acquireLock: false when the lock is held elsewhere.
func (o *Orchestrator) acquireLock(ctx context.Context, key string) bool {
	if o.locking == nil {
		return true
	}
	owner := o.lockingOwner
	return o.locking.Acquire(ctx, []string{key}, lockingtypes.AcquireArgs{Expire: o.lockDuration, OwnerID: &owner}) == nil
}

// RenewLock mirrors renewLock.
func (o *Orchestrator) RenewLock(ctx context.Context, key string) bool {
	return o.acquireLock(ctx, key)
}

// processEntity mirrors #processEntity.
func (o *Orchestrator) processEntity(ctx context.Context, taskRunner func(ctx context.Context, entity string) error, entity string) error {
	if !o.acquireLock(ctx, entity) {
		o.logger.Warn(fmt.Sprintf("[Index engine] failed to acquire lock for entity '%s' on pid %d. It means another process is already processing this entity or a lock is still present in your locking provider.", entity, os.Getpid()))
		return nil
	}
	err := taskRunner(ctx, entity)
	if err != nil {
		o.state = "error"
	}
	if o.locking != nil {
		owner := o.lockingOwner
		if _, rerr := o.locking.Release(ctx, []string{entity}, lockingtypes.ReleaseArgs{OwnerID: &owner}); rerr != nil {
			o.logger.Error(fmt.Sprintf("[Index engine] failed to release lock for entity '%s'", entity))
		}
	}
	return err
}

// Process mirrors process(taskRunner).
func (o *Orchestrator) Process(ctx context.Context, taskRunner func(ctx context.Context, entity string) error) error {
	if o.state != "idle" {
		return errors.New("Cannot re-run an already running orchestrator instance")
	}
	o.state = "processing"
	for _, entity := range o.entities {
		if entity == "" {
			o.state = "completed"
			break
		}
		if err := o.processEntity(ctx, taskRunner, entity); err != nil {
			return err
		}
	}
	o.state = "completed"
	return nil
}
