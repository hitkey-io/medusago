package composer

import "context"

// WorkflowEngineAPI is the public surface of a workflow engine module
// service (IWorkflowEngineService: run, setStepSuccess, setStepFailure,
// subscribe, unsubscribe, getRunningTransaction, cancel, retryStep) over
// the engine of a container. The workflow engine modules embed it.
type WorkflowEngineAPI struct {
	EngineContainer Container
}

// Run mirrors workflowEngine.run(workflowId, { input, transactionId, ... }).
func (a WorkflowEngineAPI) Run(ctx context.Context, workflowID string, input any, opts RunOptions) (any, RunInfo, error) {
	opts.Notify = true
	return RunByID(ctx, a.EngineContainer, workflowID, input, opts)
}

// SetStepSuccess mirrors workflowEngine.setStepSuccess.
func (a WorkflowEngineAPI) SetStepSuccess(ctx context.Context, key IdempotencyKey, response any, opts ...StepResponseOptions) error {
	return SetStepSuccess(ctx, a.EngineContainer, key, response, opts...)
}

// SetStepFailure mirrors workflowEngine.setStepFailure.
func (a WorkflowEngineAPI) SetStepFailure(ctx context.Context, key IdempotencyKey, err error, opts ...StepResponseOptions) error {
	return SetStepFailure(ctx, a.EngineContainer, key, err, opts...)
}

// Subscribe mirrors workflowEngine.subscribe.
func (a WorkflowEngineAPI) Subscribe(workflowID, transactionID, subscriberID string, fn Subscriber) {
	Subscribe(workflowID, transactionID, subscriberID, fn)
}

// Unsubscribe mirrors workflowEngine.unsubscribe.
func (a WorkflowEngineAPI) Unsubscribe(workflowID, transactionID, subscriberID string) {
	Unsubscribe(workflowID, transactionID, subscriberID)
}

// GetRunningTransaction mirrors workflowEngine.getRunningTransaction.
func (a WorkflowEngineAPI) GetRunningTransaction(ctx context.Context, workflowID, transactionID string) (*Checkpoint, error) {
	return GetRunningTransaction(ctx, a.EngineContainer, workflowID, transactionID)
}

// Cancel mirrors workflowEngine.cancel.
func (a WorkflowEngineAPI) Cancel(ctx context.Context, workflowID string, opts CancelOptions) (*CancelResult, error) {
	return Cancel(ctx, a.EngineContainer, workflowID, opts)
}

// RetryStep mirrors workflowEngine.retryStep.
func (a WorkflowEngineAPI) RetryStep(ctx context.Context, key IdempotencyKey) (*Checkpoint, error) {
	return RetryStep(ctx, a.EngineContainer, key)
}
