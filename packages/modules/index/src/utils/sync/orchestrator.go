// Package sync mirrors @medusajs/index/src/utils/sync.
package sync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	stdsync "sync"
	"time"
)

// locks mirrors the in-memory locking provider: one owner per key until
// released or expired.
var (
	locksMu stdsync.Mutex
	locks   = map[string]lockEntry{}
)

type lockEntry struct {
	owner   string
	expires time.Time
}

func acquire(key, owner string, expire time.Duration) bool {
	locksMu.Lock()
	defer locksMu.Unlock()
	if l, ok := locks[key]; ok && l.owner != owner && time.Now().Before(l.expires) {
		return false
	}
	locks[key] = lockEntry{owner: owner, expires: time.Now().Add(expire)}
	return true
}

func release(key, owner string) error {
	locksMu.Lock()
	defer locksMu.Unlock()
	if l, ok := locks[key]; ok && l.owner != owner {
		return errors.New("lock owned by another owner")
	}
	delete(locks, key)
	return nil
}

// Orchestrator mirrors Orchestrator: processes the entities one by one,
// each under a lock.
type Orchestrator struct {
	logger       *slog.Logger
	lockingOwner string
	state        string
	lockDuration time.Duration
	entities     []string
}

// NewOrchestrator mirrors new Orchestrator(lockingModule, entities, options).
func NewOrchestrator(entities []string, lockDuration time.Duration, logger *slog.Logger) *Orchestrator {
	return &Orchestrator{logger: logger, lockingOwner: fmt.Sprintf("index-sync-%d", os.Getpid()), state: "idle", lockDuration: lockDuration, entities: entities}
}

// RenewLock mirrors renewLock.
func (o *Orchestrator) RenewLock(key string) bool {
	return acquire(key, o.lockingOwner, o.lockDuration)
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
		if !acquire(entity, o.lockingOwner, o.lockDuration) {
			o.logger.Warn(fmt.Sprintf("[Index engine] failed to acquire lock for entity '%s' on pid %d. It means another process is already processing this entity or a lock is still present in your locking provider.", entity, os.Getpid()))
			continue
		}
		err := taskRunner(ctx, entity)
		if rerr := release(entity, o.lockingOwner); rerr != nil {
			o.logger.Error(fmt.Sprintf("[Index engine] failed to release lock for entity '%s'", entity))
		}
		if err != nil {
			o.state = "error"
			return err
		}
	}
	o.state = "completed"
	return nil
}
