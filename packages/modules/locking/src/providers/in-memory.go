// Package providers mirrors @medusajs/locking/src/providers: the built-in
// in-memory locking provider.
package providers

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
)

// InMemoryIdentifier mirrors InMemoryLockingProvider.identifier.
const InMemoryIdentifier = "in-memory"

type lockInfo struct {
	ownerID *string
	// expiration in unix milliseconds (0: null)
	expiration int64
	// currentPromise: closed when the lock is released or taken over
	currentPromise chan struct{}
	resolved       bool
}

func (l *lockInfo) resolve() {
	if l.currentPromise != nil && !l.resolved {
		l.resolved = true
		close(l.currentPromise)
	}
}

// InMemoryLockingProvider mirrors InMemoryLockingProvider: process-local
// locks, released explicitly or when their expiration passed.
type InMemoryLockingProvider struct {
	mu    sync.Mutex
	locks map[string]*lockInfo
}

// NewInMemoryLockingProvider mirrors new InMemoryLockingProvider().
func NewInMemoryLockingProvider() *InMemoryLockingProvider {
	return &InMemoryLockingProvider{locks: map[string]*lockInfo{}}
}

// ExecuteTimeout mirrors `Math.max(args?.timeout ?? fallback, 1)` (NaN: 1).
func ExecuteTimeout(timeout *float64, fallback float64) float64 {
	t := fallback
	if timeout != nil {
		t = *timeout
	}
	t = math.Max(t, 1)
	if math.IsNaN(t) {
		return 1
	}
	return t
}

// SecondsDuration converts JS seconds to a duration.
func SecondsDuration(s float64) time.Duration {
	return time.Duration(s * float64(time.Second))
}

// Execute mirrors execute: the job runs while the keys are held; the keys
// are awaited (queue) for at most timeout seconds.
func (p *InMemoryLockingProvider) Execute(ctx context.Context, keys []string, job lockingtypes.Job, args lockingtypes.ExecuteArgs) (any, error) {
	timeoutSeconds := ExecuteTimeout(args.Timeout, 5)
	// Unique per call, so the release below can only delete the lock this
	// call took.
	ownerID := "execute:" + RandomUUID()

	cancelled := &atomic.Bool{}
	acquired := make(chan error, 1)
	go func() {
		acquired <- p.acquire(keys, &ownerID, 0, true, cancelled)
	}()
	timer := time.NewTimer(SecondsDuration(timeoutSeconds))
	select {
	case err := <-acquired:
		timer.Stop()
		if err != nil {
			return nil, err
		}
	case <-timer.C:
		cancelled.Store(true)
		// the queued acquisition can still take the keys: give them back
		go func() {
			if err := <-acquired; err == nil {
				_, _ = p.Release(context.Background(), keys, lockingtypes.ReleaseArgs{OwnerID: &ownerID})
			}
		}()
		return nil, errors.New("Timed-out acquiring lock.")
	}
	defer func() {
		_, _ = p.Release(context.Background(), keys, lockingtypes.ReleaseArgs{OwnerID: &ownerID})
	}()
	return job(ctx)
}

// Acquire mirrors acquire.
func (p *InMemoryLockingProvider) Acquire(ctx context.Context, keys []string, args lockingtypes.AcquireArgs) error {
	return p.acquire(keys, args.OwnerID, args.Expire, args.AwaitQueue, nil)
}

func sameOwner(a, b *string) bool { return a != nil && b != nil && *a == *b }

// acquire mirrors acquire_.
func (p *InMemoryLockingProvider) acquire(keys []string, ownerID *string, expire float64, awaitQueue bool, cancelled *atomic.Bool) error {
	for {
		var wait chan struct{}
		p.mu.Lock()
		var err error
		for _, key := range keys {
			lock := p.locks[key]
			now := time.Now().UnixMilli()
			newLock := func() *lockInfo {
				l := &lockInfo{ownerID: ownerID, currentPromise: make(chan struct{})}
				if expire != 0 {
					l.expiration = now + int64(expire*1000)
				}
				return l
			}
			if lock == nil {
				p.locks[key] = newLock()
				continue
			}
			if lock.expiration != 0 && lock.expiration <= now {
				lock.resolve()
				p.locks[key] = newLock()
				continue
			}
			if lock.ownerID != nil && sameOwner(lock.ownerID, ownerID) {
				if expire != 0 {
					lock.expiration = now + int64(expire*1000)
				}
				continue
			}
			if lock.currentPromise != nil && awaitQueue {
				wait = lock.currentPromise
				break
			}
			err = fmt.Errorf(`Failed to acquire lock for key "%s"`, key)
			break
		}
		p.mu.Unlock()
		if err != nil {
			return err
		}
		if wait == nil {
			return nil
		}
		<-wait
		if cancelled != nil && cancelled.Load() {
			return nil
		}
	}
}

// Release mirrors release: a lock is deleted when the owner matches (any
// owner when undefined).
func (p *InMemoryLockingProvider) Release(ctx context.Context, keys []string, args lockingtypes.ReleaseArgs) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	success := true
	for _, key := range keys {
		lock := p.locks[key]
		if lock == nil {
			success = false
			continue
		}
		if args.OwnerID != nil && !sameOwner(lock.ownerID, args.OwnerID) {
			success = false
			continue
		}
		lock.resolve()
		delete(p.locks, key)
	}
	return success, nil
}

// ReleaseAll mirrors releaseAll.
func (p *InMemoryLockingProvider) ReleaseAll(ctx context.Context, args lockingtypes.ReleaseArgs) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for key, lock := range p.locks {
		if args.OwnerID == nil || sameOwner(lock.ownerID, args.OwnerID) {
			lock.resolve()
			delete(p.locks, key)
		}
	}
	return nil
}
