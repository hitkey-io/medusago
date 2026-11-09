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
	if !ok {
		return nil
	}
	step := CreateStep(w.ID+"."+name, h.invoke, h.compensate)
	_, err := step.Run(e, input)
	return err
}

// CreateHook mirrors createHook(name, input) for the workflow being
// executed (the body does not need a reference to its workflow).
func CreateHook(e *Execution, name string, input any) error {
	if e.hooks == nil {
		return nil
	}
	e.hooks.mu.RLock()
	h, ok := e.hooks.handlers[name]
	e.hooks.mu.RUnlock()
	if !ok {
		return nil
	}
	step := CreateStep(e.workflowID+"."+name, h.invoke, h.compensate)
	_, err := step.Run(e, input)
	return err
}
