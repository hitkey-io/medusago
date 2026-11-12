package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// Workflow engine operations on a stored transaction: getRunningTransaction,
// cancel and retryStep (workflow-orchestrator.ts and
// TransactionOrchestrator.cancelTransaction / retryStep).
//
// A transaction is reached the way the engine storage get(key) reaches it:
// a run of this process (the in-memory engine keeps the waiting steps of
// its runs) or the stored checkpoint (workflow_execution row, the Redis
// checkpoint). Only stored transactions exist for the engine: a workflow
// without store / retention / async steps has no checkpoint.

// errRunCancelled aborts the body of a cancelled transaction: the steps
// that ran are compensated. It is not an invoke error.
var errRunCancelled = errors.New("transaction cancelled")

// CheckpointReader is implemented by a workflow engine module: the latest
// stored checkpoint of a transaction whatever its state (nil when none).
type CheckpointReader interface {
	ReadCheckpoint(ctx context.Context, workflowID, transactionID string) (*Checkpoint, error)
}

var (
	liveMu sync.Mutex
	live   = map[string]*Execution{}
)

func liveKey(workflowID, transactionID string) string { return workflowID + ":" + transactionID }

// registerLive records a root run of this process until it finishes.
func (e *Execution) registerLive() {
	e.finished = make(chan struct{})
	liveMu.Lock()
	live[liveKey(e.rootWorkflowID, e.transactionID)] = e
	liveMu.Unlock()
}

// markFinished closes the finished channel and forgets the run.
func (e *Execution) markFinished() {
	if e.finished == nil {
		return
	}
	e.finishedOnce.Do(func() {
		liveMu.Lock()
		k := liveKey(e.rootWorkflowID, e.transactionID)
		if live[k] == e {
			delete(live, k)
		}
		liveMu.Unlock()
		close(e.finished)
	})
}

// forgetLive forgets a suspended run (its checkpoint holds it).
func (e *Execution) forgetLive() {
	liveMu.Lock()
	k := liveKey(e.rootWorkflowID, e.transactionID)
	if live[k] == e {
		delete(live, k)
	}
	liveMu.Unlock()
}

func liveExecution(workflowID, transactionID string) *Execution {
	liveMu.Lock()
	defer liveMu.Unlock()
	return live[liveKey(workflowID, transactionID)]
}

func (e *Execution) root() *Execution {
	r := e
	for r.parentExec != nil {
		r = r.parentExec
	}
	return r
}

func (e *Execution) isCancelled() bool {
	r := e.root()
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cancelled
}

// checkpoint is the transaction of a run of this process.
func (e *Execution) checkpoint() *Checkpoint {
	flow := toOrderedRecord(e.journal.flow(e.rootWorkflowID, e.transactionID, e.eventGroupID, e.rootOptions))
	ctx := toOrderedRecord(e.journal.context())
	data, _ := ctx.Get("data")
	errs, _ := ctx.Get("errors")
	cp := &Checkpoint{Flow: flow}
	cp.Context, _ = data.(*dal.Record)
	cp.Errors, _ = errs.([]any)
	return cp
}

func toOrderedRecord(v any) *dal.Record {
	b, err := json.Marshal(v)
	if err != nil {
		return dal.NewRecord()
	}
	out, err := dal.DecodeOrderedJSON(b)
	if err != nil {
		return dal.NewRecord()
	}
	r, _ := out.(*dal.Record)
	if r == nil {
		return dal.NewRecord()
	}
	return r
}

// stored reports a run the engine stores (store / retention / async steps).
func (e *Execution) stored() bool {
	e.journal.mu.Lock()
	defer e.journal.mu.Unlock()
	return e.rootOptions.Store || e.rootOptions.RetentionTime > 0 || e.journal.hasAsync
}

func flowState(cp *Checkpoint) string { return asStr(recGet(cp.Flow, "state")) }

// loadTransaction mirrors storage.get(key, { isCancelling }): a done
// transaction is only returned when cancelling, a failed or reverted one
// never.
func loadTransaction(ctx context.Context, c Container, workflowID, transactionID string, cancelling bool) (*Checkpoint, *Execution, error) {
	var cp *Checkpoint
	e := liveExecution(workflowID, transactionID)
	if e != nil && e.stored() {
		cp = e.checkpoint()
	} else {
		e = nil
		reader, _ := c.Resolve(utilsmodulessdk.WORKFLOW_ENGINE).(CheckpointReader)
		if reader == nil {
			return nil, nil, nil
		}
		var err error
		if cp, err = reader.ReadCheckpoint(ctx, workflowID, transactionID); err != nil || cp == nil {
			return nil, nil, err
		}
	}
	switch flowState(cp) {
	case TransactionStateFailed, TransactionStateReverted:
		return nil, nil, nil
	case TransactionStateDone:
		if !cancelling {
			return nil, nil, nil
		}
	}
	return cp, e, nil
}

func transactionNotFound(transactionID string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf("Transaction %s could not be found.", transactionID))
}

func workflowNotFound(workflowID string) error {
	return fmt.Errorf("Workflow with id \"%s\" not found.", workflowID)
}

// GetRunningTransaction mirrors workflowEngine.getRunningTransaction.
func GetRunningTransaction(ctx context.Context, c Container, workflowID, transactionID string) (*Checkpoint, error) {
	if workflowID == "" {
		return nil, errors.New("Workflow ID is required")
	}
	if transactionID == "" {
		return nil, errors.New("TransactionId ID is required")
	}
	if !IsRegisteredWorkflow(workflowID) {
		return nil, workflowNotFound(workflowID)
	}
	cp, _, err := loadTransaction(ctx, c, workflowID, transactionID, false)
	if err != nil {
		return nil, err
	}
	if cp == nil {
		return nil, transactionNotFound(transactionID)
	}
	return cp, nil
}

// CancelOptions are the options of workflowEngine.cancel.
type CancelOptions struct {
	TransactionID string
	// ThrowOnError defaults to true. An unknown transaction throws
	// whatever its value (getRunningTransaction throws first).
	ThrowOnError *bool
}

// CancelResult is what workflowEngine.cancel returns: the acknowledgement
// and the transaction.
type CancelResult struct {
	Acknowledgement *dal.Record
	Transaction     *Checkpoint
}

// Cancel mirrors workflowEngine.cancel(workflowId, { transactionId }):
// cancelTransaction sets the flow WAITING_TO_COMPENSATE; the steps that ran
// are compensated at once, or once the waiting step of the transaction
// responds.
func Cancel(ctx context.Context, c Container, workflowID string, opts CancelOptions) (*CancelResult, error) {
	if workflowID == "" {
		return nil, errors.New("Workflow ID is required")
	}
	if opts.TransactionID == "" {
		return nil, errors.New("Transaction ID is required")
	}
	if !IsRegisteredWorkflow(workflowID) {
		if durableEngine(c) != nil {
			// the Redis engine throws a MedusaError here
			return nil, notFoundWorkflow(workflowID)
		}
		return nil, workflowNotFound(workflowID)
	}
	transactionID := opts.TransactionID
	throwOnError := opts.ThrowOnError == nil || *opts.ThrowOnError
	var unlock func()
	if de := durableEngine(c); de != nil {
		var err error
		if unlock, err = de.LockTransaction(ctx, workflowID, transactionID); err != nil {
			return nil, err
		}
		defer unlock()
	}
	cp, e, err := loadTransaction(ctx, c, workflowID, transactionID, true)
	if err != nil {
		return nil, err
	}
	if cp == nil {
		// getRunningTransaction throws before throwOnError applies
		return nil, transactionNotFound(transactionID)
	}
	switch flowState(cp) {
	case TransactionStateFailed:
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Cannot revert a permanent failed transaction.")
	case TransactionStateCompensating, TransactionStateWaitingToCompensate:
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, "Cannot revert a transaction that is already compensating.")
	}

	finished := false
	state := TransactionStateWaitingToCompensate
	switch {
	case e != nil:
		// a run of this process: it compensates once its running or
		// waiting step finishes
		e.mu.Lock()
		e.cancelled = true
		e.mu.Unlock()
		e.journal.mu.Lock()
		e.journal.cancelledAt = nowMillis()
		e.journal.rev++
		e.journal.mu.Unlock()
		e.journal.setState(TransactionStateWaitingToCompensate)
		e.persist(TransactionStateWaitingToCompensate)
		cp = e.checkpoint()
	case hasPendingStep(cp):
		// a step waits (or runs elsewhere): the checkpoint is marked and
		// the response of the step compensates the transaction
		flow := cp.Flow
		flow.Set("state", TransactionStateWaitingToCompensate)
		flow.Set("cancelledAt", nowMillis())
		if store := executionStore(c); store != nil {
			row := ExecutionRow{WorkflowID: workflowID, TransactionID: transactionID, RunID: asStr(recGet(flow, "runId")),
				State: TransactionStateWaitingToCompensate, Execution: flow,
				Context: map[string]any{"data": cp.Context, "errors": cp.Errors}}
			if rt := int(asNum(recGet(asRec(recGet(flow, "options")), "retentionTime"))); rt > 0 {
				row.RetentionTime = &rt
			}
			if err := store.SaveExecution(context.WithoutCancel(ctx), row); err != nil {
				return nil, err
			}
		}
	default:
		// every step finished: the transaction is compensated now. Upstream
		// the transaction comes from getRunningTransaction, which builds the
		// workflow without a container (exportedWorkflow()): its
		// compensations run in an empty container whatever the cancel
		// options, so those resolving a module fail and the transaction
		// ends FAILED. Only the workflow engine (its storage) is reachable.
		rs := parseCheckpoint(cp)
		rp := newReplayState(rs, "", nil)
		rp.cancel = true
		run, err := resumeRun(ctx, engineOnlyContainer{c}, rs, rp)
		if err != nil {
			return nil, err
		}
		select {
		case <-run.finished:
			finished = true
			state = run.state
		case <-run.detached:
		}
		if finished {
			cp.Flow.Set("state", state)
		}
	}
	if finished && state == TransactionStateFailed && throwOnError {
		// FAILED without invoke errors: workflow-export throws
		// "Unknown error"
		return nil, errors.New("Unknown error")
	}
	ack := dal.NewRecord()
	ack.Set("transactionId", transactionID)
	ack.Set("workflowId", workflowID)
	if md := asRec(recGet(cp.Flow, "metadata")); md != nil {
		if v, ok := md.Get("parentStepIdempotencyKey"); ok && v != nil {
			ack.Set("parentStepIdempotencyKey", v)
		}
	}
	ack.Set("hasFinished", finished)
	ack.Set("hasFailed", state == TransactionStateFailed)
	ack.Set("exists", true)
	return &CancelResult{Acknowledgement: ack, Transaction: cp}, nil
}

// hasPendingStep reports a step that is invoking (waiting for a response
// or running).
func hasPendingStep(cp *Checkpoint) bool {
	for _, st := range parseCheckpoint(cp).steps {
		if st.invokeState == stepStateInvoking {
			return true
		}
	}
	return false
}

// cancelledResponseError is what setStepSuccess / setStepFailure answer
// for a cancelled transaction compensated after the response: reverted
// without invoke errors -> "Unknown error" (setStepFailure: none).
func cancelledResponseError(failure bool, state string) error {
	if failure && state == TransactionStateReverted {
		return nil
	}
	if state == TransactionStateReverted || state == TransactionStateFailed {
		return errors.New("Unknown error")
	}
	return nil
}

// RetryStep mirrors workflowEngine.retryStep({ idempotencyKey }): a step
// waiting for a response is invoked again (its handler runs; an empty
// response of an async step waits again). It returns the transaction as
// it is when the retry is scheduled.
func RetryStep(ctx context.Context, c Container, key IdempotencyKey) (*Checkpoint, error) {
	if !IsRegisteredWorkflow(key.WorkflowID) {
		return nil, workflowNotFound(key.WorkflowID)
	}
	cp, e, err := loadTransaction(ctx, c, key.WorkflowID, key.TransactionID, false)
	if err != nil {
		return nil, err
	}
	if cp == nil {
		return nil, transactionNotFound(key.TransactionID)
	}
	st := parseCheckpoint(cp).stepByAction(key.StepID)
	if st == nil {
		return nil, errors.New("Action not found.")
	}
	if key.Action != "" && key.Action != "invoke" {
		return nil, errors.New("Incorrect action type.")
	}
	if !st.waiting() {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.NOT_ALLOWED, fmt.Sprintf("Cannot retry step when status is %s", st.invokeStatus))
	}
	// the retry runs after the response (scheduleRetry(step, 0))
	if e != nil {
		coord := stepCoordinator(c)
		go func() { _ = coord.Deliver(context.WithoutCancel(ctx), key, StepResult{Retry: true}) }()
		return cp, nil
	}
	if de := durableEngine(c); de != nil {
		go func() {
			ctx := context.WithoutCancel(ctx)
			unlock, err := de.LockTransaction(ctx, key.WorkflowID, key.TransactionID)
			if err != nil {
				return
			}
			defer unlock()
			stored, err := de.LoadCheckpoint(ctx, key.WorkflowID, key.TransactionID)
			if err != nil || stored == nil {
				return
			}
			rs := parseCheckpoint(stored)
			if s := rs.stepByAction(key.StepID); s == nil || !s.waiting() {
				return
			}
			_ = startResume(ctx, c, rs, newReplayState(rs, key.StepID, &StepResult{Retry: true}))
		}()
	}
	return cp, nil
}

// engineOnlyContainer resolves the workflow engine only (the container of a
// workflow built without one, see Cancel).
type engineOnlyContainer struct{ c Container }

func (e engineOnlyContainer) Resolve(key string) any {
	if key == utilsmodulessdk.WORKFLOW_ENGINE {
		return e.c.Resolve(key)
	}
	return nil
}
