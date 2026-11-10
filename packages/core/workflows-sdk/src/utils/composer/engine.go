package composer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// WorkflowOptions mirrors the createWorkflow({ name, store, retentionTime,
// idempotent }) options.
type WorkflowOptions struct {
	Store bool
	// RetentionTime is in seconds; a finished execution is kept that long.
	RetentionTime int
	Idempotent    bool
}

// ExecutionRow is one workflow_execution row.
type ExecutionRow struct {
	WorkflowID    string
	TransactionID string
	RunID         string
	State         string
	Execution     any
	Context       any
	RetentionTime *int
}

// ExecutionStore persists workflow executions (the workflow engine module's
// InMemoryDistributedTransactionStorage DB side).
type ExecutionStore interface {
	SaveExecution(ctx context.Context, row ExecutionRow) error
	DeleteExecution(ctx context.Context, workflowID, transactionID, runID string) error
}

// RunInfo describes a run like the returned DistributedTransaction.
type RunInfo struct {
	TransactionID string
	RunID         string
	State         string
	// Detached reports a run that returned before finishing (a background
	// or async step); it completes in the background.
	Detached bool
}

func executionStore(c Container) ExecutionStore {
	if c == nil {
		return nil
	}
	s, _ := c.Resolve(utilsmodulessdk.WORKFLOW_ENGINE).(ExecutionStore)
	return s
}

// persist writes the current checkpoint (saveToDb).
func (e *Execution) persist(state string) {
	store := executionStore(e.container)
	if store == nil {
		return
	}
	e.journal.setState(state)
	row := ExecutionRow{
		WorkflowID: e.rootWorkflowID, TransactionID: e.transactionID, RunID: e.journal.runID, State: state,
		Execution: e.journal.flow(e.rootWorkflowID, e.transactionID, e.eventGroupID, e.rootOptions),
		Context:   e.journal.context(),
	}
	if e.rootOptions.RetentionTime > 0 {
		rt := e.rootOptions.RetentionTime
		row.RetentionTime = &rt
	}
	if err := store.SaveExecution(context.WithoutCancel(e.ctx), row); err != nil {
		slog.Error(fmt.Sprintf("failed to save workflow execution %s:%s: %v", e.rootWorkflowID, e.transactionID, err))
	}
}

// removeExecution deletes the stored execution (deleteFromDb).
func (e *Execution) removeExecution() {
	store := executionStore(e.container)
	if store == nil {
		return
	}
	if err := store.DeleteExecution(context.WithoutCancel(e.ctx), e.rootWorkflowID, e.transactionID, e.journal.runID); err != nil {
		slog.Error(fmt.Sprintf("failed to delete workflow execution %s:%s: %v", e.rootWorkflowID, e.transactionID, err))
	}
}

// stepResult is what setStepSuccess / setStepFailure deliver to a waiting
// async step.
type stepResult struct {
	output any
	err    error
}

var (
	waitersMu sync.Mutex
	waiters   = map[string]chan stepResult{}
)

func waiterKey(workflowID, transactionID, stepID string) string {
	return workflowID + ":" + transactionID + ":" + stepID
}

func registerWaiter(key string) chan stepResult {
	ch := make(chan stepResult, 1)
	waitersMu.Lock()
	waiters[key] = ch
	waitersMu.Unlock()
	return ch
}

func takeWaiter(key string) chan stepResult {
	waitersMu.Lock()
	defer waitersMu.Unlock()
	ch := waiters[key]
	delete(waiters, key)
	return ch
}

// IdempotencyKey mirrors the { action, transactionId, stepId, workflowId }
// key of setStepSuccess / setStepFailure.
type IdempotencyKey struct {
	Action        string
	TransactionID string
	StepID        string
	WorkflowID    string
}

var (
	registryMu sync.RWMutex
	registry   = map[string]bool{}
)

// registerWorkflow mirrors WorkflowManager.register (ids only).
func registerWorkflow(id string) {
	registryMu.Lock()
	registry[id] = true
	registryMu.Unlock()
}

// IsRegisteredWorkflow reports whether a workflow with this id exists.
func IsRegisteredWorkflow(id string) bool {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return registry[id]
}

// notFoundWaiting mirrors the errors of registerStepSuccess for a step that
// is not waiting in this process: an unknown workflow, else an unknown
// transaction.
func notFoundWaiting(key IdempotencyKey) error {
	if !IsRegisteredWorkflow(key.WorkflowID) {
		return fmt.Errorf("Workflow with id \"%s\" not found.", key.WorkflowID)
	}
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Transaction %s could not be found.", key.TransactionID))
}

// SetStepSuccess mirrors workflowEngine.setStepSuccess: the waiting async
// step completes with the response and the workflow continues.
func SetStepSuccess(key IdempotencyKey, response any) error {
	ch := takeWaiter(waiterKey(key.WorkflowID, key.TransactionID, key.StepID))
	if ch == nil {
		return notFoundWaiting(key)
	}
	ch <- stepResult{output: response}
	return nil
}

// SetStepFailure mirrors workflowEngine.setStepFailure.
func SetStepFailure(key IdempotencyKey, err error) error {
	ch := takeWaiter(waiterKey(key.WorkflowID, key.TransactionID, key.StepID))
	if ch == nil {
		return notFoundWaiting(key)
	}
	ch <- stepResult{err: err}
	return nil
}

// convertOutput converts a value delivered by setStepSuccess (JSON-like) to
// the step output type.
func convertOutput[O any](v any) (O, error) {
	var out O
	if v == nil {
		return out, nil
	}
	if o, ok := v.(O); ok {
		return o, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(b, &out)
	return out, err
}
