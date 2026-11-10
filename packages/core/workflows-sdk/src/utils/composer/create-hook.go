package composer

import "sync"

// HookHandler is a hook handler: a step whose input is the hook input.
type HookHandler struct {
	invoke     func(StepExecutionContext, any) (StepResponse[any], error)
	compensate func(StepExecutionContext, any) error
}

type hookRegistry struct {
	mu       sync.RWMutex
	handlers map[string]HookHandler
}

func newHookRegistry() *hookRegistry { return &hookRegistry{handlers: map[string]HookHandler{}} }

// RegisterHook mirrors workflow.hooks.<name>(handler, compensate?): one
// handler per hook, the last registration wins.
func (w *Workflow[I, O]) RegisterHook(name string, invoke func(StepExecutionContext, any) (StepResponse[any], error), compensate func(StepExecutionContext, any) error) {
	w.hooks.mu.Lock()
	defer w.hooks.mu.Unlock()
	w.hooks.handlers[name] = HookHandler{invoke: invoke, compensate: compensate}
}

// CreateHook mirrors createHook(name, input) inside the workflow body: the
// registered handler runs as the step "<workflowId>.<name>"; without a
// handler it is a no-op.
func (w *Workflow[I, O]) CreateHook(e *Execution, name string, input any) error {
	w.hooks.mu.RLock()
	h, ok := w.hooks.handlers[name]
	w.hooks.mu.RUnlock()
	_, err := runHook(e, w.ID, name, h, ok, input)
	return err
}

// runHook runs the hook as the step "<workflowId>.<name>" (flow action
// <name>); without a handler the hook step completes with no output.
func runHook(e *Execution, workflowID, name string, h HookHandler, ok bool, input any) (any, error) {
	if !ok {
		if e.journal != nil {
			rec := e.journal.begin(name, StepOptions{}, true)
			e.journal.attempt(rec, 1)
			e.journal.succeed(rec, nil, false, nil, false)
		}
		return nil, nil
	}
	step := CreateStep(workflowID+"."+name, h.invoke, h.compensate)
	step.action = name
	return step.Run(e, input)
}

// CreateHook mirrors createHook(name, input) for the workflow being
// executed (the body does not need a reference to its workflow).
func CreateHook(e *Execution, name string, input any) error {
	var h HookHandler
	ok := false
	if e.hooks != nil {
		e.hooks.mu.RLock()
		h, ok = e.hooks.handlers[name]
		e.hooks.mu.RUnlock()
	}
	_, err := runHook(e, e.workflowID, name, h, ok, input)
	return err
}

// CreateHookWithResult mirrors createHook(name, input).getResult(): the
// handler's output (nil without a handler).
func CreateHookWithResult(e *Execution, name string, input any) (any, error) {
	var h HookHandler
	ok := false
	if e.hooks != nil {
		e.hooks.mu.RLock()
		h, ok = e.hooks.handlers[name]
		e.hooks.mu.RUnlock()
	}
	return runHook(e, e.workflowID, name, h, ok, input)
}
