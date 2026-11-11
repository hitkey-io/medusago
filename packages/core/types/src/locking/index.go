// Package locking mirrors @medusajs/types/src/locking: the locking module
// and locking provider interfaces.
package locking

import "context"

// Job is the function run while the keys are held. Its context is cancelled
// when the provider loses the lock while the job runs (the AbortSignal
// passed to the job upstream).
type Job func(ctx context.Context) (any, error)

// ExecuteArgs mirrors the args of execute. Nil fields are undefined.
type ExecuteArgs struct {
	// Timeout is the time (seconds) to wait for the keys.
	Timeout *float64
	// Expire is the lease (seconds) of the keys (Redis provider).
	Expire *float64
	// Provider selects the provider (module only; default provider when
	// empty).
	Provider string
}

// AcquireArgs mirrors the args of acquire.
type AcquireArgs struct {
	// OwnerID is the owner of the lock (nil: undefined).
	OwnerID *string
	// Expire is the expiration (seconds) of the lock; 0 is undefined.
	Expire float64
	// Provider selects the provider (module only).
	Provider string
	// AwaitQueue waits for the keys instead of failing (providers only,
	// used by execute).
	AwaitQueue bool
}

// ReleaseArgs mirrors the args of release and releaseAll.
type ReleaseArgs struct {
	// OwnerID is the owner of the lock (nil: undefined).
	OwnerID *string
	// Provider selects the provider (module only).
	Provider string
}

// ILockingProvider mirrors ILockingProvider.
type ILockingProvider interface {
	Execute(ctx context.Context, keys []string, job Job, args ExecuteArgs) (any, error)
	Acquire(ctx context.Context, keys []string, args AcquireArgs) error
	Release(ctx context.Context, keys []string, args ReleaseArgs) (bool, error)
	ReleaseAll(ctx context.Context, args ReleaseArgs) error
}

// ILockingModule mirrors ILockingModule: the Provider field of the args
// selects the provider (the default one when empty).
type ILockingModule interface {
	ILockingProvider
}

// Shutdowner is a provider holding resources released on shutdown.
type Shutdowner interface {
	Shutdown() error
}

// Owner returns a pointer to the owner id (helper for the args).
func Owner(id string) *string { return &id }

// Seconds returns a pointer to a number of seconds (helper for the args).
func Seconds(s float64) *float64 { return &s }
