package composer

import "sync"

// HookHandler is a hook handler: a step whose input is the hook input.
type HookHandler struct {
	invoke     func(StepExecutionContext, any) (StepResponse[any], error)
	compensate func(StepExecutionContext, any) error
	// enabled restricts the handler to the applications it belongs to
	// (a plugin's handler: the applications using the plugin)
	enabled func(Container) bool
}

func (h HookHandler) activeIn(c Container) bool { return h.enabled == nil || h.enabled(c) }

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

// RegisterHookWhen is RegisterHook for the runs whose container satisfies
// enabled (a plugin registers its handlers for the applications that use
// it; elsewhere the hook has no handler).
func (w *Workflow[I, O]) RegisterHookWhen(name string, enabled func(Container) bool, invoke func(StepExecutionContext, any) (StepResponse[any], error), compensate func(StepExecutionContext, any) error) {
	w.hooks.mu.Lock()
	defer w.hooks.mu.Unlock()
	w.hooks.handlers[name] = HookHandler{invoke: invoke, compensate: compensate, enabled: enabled}
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
// <name>); without a handler the hook step completes with no output. Like
// upstream, a hook is always compensated (a no-op without a compensation).
func runHook(e *Execution, workflowID, name string, h HookHandler, ok bool, input any) (any, error) {
	if ok && !h.activeIn(e.container) {
		ok = false
	}
	if !ok {
		if e.journal == nil {
			return nil, nil
		}
		rec := e.noopStep(name, func(rec *stepRecord) {
			e.journal.succeed(rec, nil, false, nil, false)
		})
		e.pushNoopCompensation(name, rec)
		return nil, nil
	}
	comp := h.compensate
	if comp == nil {
		comp = func(StepExecutionContext, any) error { return nil }
	}
	step := CreateStep(workflowID+"."+name, h.invoke, comp)
	step.action = name
	return step.Run(e, input)
}

// noopStep journals a step that runs no handler of its own (a hook
// without a handler, a hook result): restored from the checkpoint by a
// replay, recorded with done otherwise.
func (e *Execution) noopStep(action string, done func(rec *stepRecord)) *stepRecord {
	if e.replay != nil {
		if st := e.replay.take(action); st != nil && st.invokeState == stepStateDone {
			return e.journal.restore(action, StepOptions{}, false, st)
		}
	}
	rec := e.journal.begin(action, StepOptions{}, false)
	e.journal.attempt(rec, 1)
	done(rec)
	return rec
}

// pushNoopCompensation registers the compensation of a step whose
// compensation does nothing (() => void 0): it runs and is recorded.
func (e *Execution) pushNoopCompensation(name string, rec *stepRecord) {
	e.pushCompensation(compensation{name: name, run: func() error {
		e.journal.compensateBegin(rec)
		e.journal.compensated(rec, nil, false)
		return nil
	}})
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
	out, err := runHook(e, e.workflowID, name, h, ok, input)
	if err != nil || e.journal == nil {
		return out, err
	}
	// getResult() is the step "get-<name>-result": its output is the hook
	// result as is (a StepResponse with no output without one)
	resultName := "get-" + name + "-result"
	rec := e.noopStep(resultName, func(rec *stepRecord) {
		if out == nil {
			e.journal.succeed(rec, nil, false, nil, false)
			return
		}
		e.journal.succeedRaw(rec, out)
	})
	e.pushNoopCompensation(resultName, rec)
	return out, nil
}
