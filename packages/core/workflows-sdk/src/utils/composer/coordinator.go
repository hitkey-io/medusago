package composer

import (
	"context"
	"sync"

	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// StepResult is what setStepSuccess / setStepFailure deliver to a waiting
// async step.
type StepResult struct {
	Output any
	Err    error
}

// StepCoordinator is implemented by the workflow engine module: it holds
// the async steps waiting for setStepSuccess / setStepFailure and delivers
// the responses to them. The in-memory engine keeps them in this process
// (LocalStepCoordinator, like its timers); the Redis engine also reaches
// the steps waiting in other instances.
type StepCoordinator interface {
	// Wait registers a waiting step; its response arrives on the channel.
	Wait(key IdempotencyKey) <-chan StepResult
	// Cancel forgets a waiting step (step timeout).
	Cancel(key IdempotencyKey)
	// Deliver delivers a response; NotFoundWaiting when no step waits.
	Deliver(ctx context.Context, key IdempotencyKey, r StepResult) error
}

// LocalStepCoordinator keeps the waiting steps of this process.
type LocalStepCoordinator struct {
	mu      sync.Mutex
	waiters map[string]chan StepResult
}

// NewLocalStepCoordinator builds a LocalStepCoordinator.
func NewLocalStepCoordinator() *LocalStepCoordinator {
	return &LocalStepCoordinator{waiters: map[string]chan StepResult{}}
}

// WaiterKey identifies a waiting step: workflow, transaction and step.
func WaiterKey(key IdempotencyKey) string {
	return key.WorkflowID + ":" + key.TransactionID + ":" + key.StepID
}

// Wait implements StepCoordinator.
func (l *LocalStepCoordinator) Wait(key IdempotencyKey) <-chan StepResult {
	ch := make(chan StepResult, 1)
	l.mu.Lock()
	l.waiters[WaiterKey(key)] = ch
	l.mu.Unlock()
	return ch
}

// Cancel implements StepCoordinator.
func (l *LocalStepCoordinator) Cancel(key IdempotencyKey) { l.take(WaiterKey(key)) }

func (l *LocalStepCoordinator) take(k string) chan StepResult {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := l.waiters[k]
	delete(l.waiters, k)
	return ch
}

// DeliverLocal delivers to a step waiting in this process; false when none
// waits.
func (l *LocalStepCoordinator) DeliverLocal(key IdempotencyKey, r StepResult) bool {
	ch := l.take(WaiterKey(key))
	if ch == nil {
		return false
	}
	ch <- r
	return true
}

// IsWaiting reports whether the step waits in this process.
func (l *LocalStepCoordinator) IsWaiting(key IdempotencyKey) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, ok := l.waiters[WaiterKey(key)]
	return ok
}

// Deliver implements StepCoordinator.
func (l *LocalStepCoordinator) Deliver(_ context.Context, key IdempotencyKey, r StepResult) error {
	if l.DeliverLocal(key, r) {
		return nil
	}
	return NotFoundWaiting(key)
}

// defaultCoordinator serves runs whose container has no workflow engine
// module (tests, scripts).
var defaultCoordinator = NewLocalStepCoordinator()

func stepCoordinator(c Container) StepCoordinator {
	if c != nil {
		if sc, ok := c.Resolve(utilsmodulessdk.WORKFLOW_ENGINE).(StepCoordinator); ok {
			return sc
		}
	}
	return defaultCoordinator
}
