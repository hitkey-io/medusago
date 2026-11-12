package composer

// When mirrors a named when(name, input, condition).then(fn) whose then
// returns a value: the steps then calls run when the condition holds, and
// upstream records the value then returns as the output of a step named
// after the when (it is skipped, with the steps of then, when the condition
// does not hold). It returns the value then returned (the zero value when
// the condition does not hold).
func When[T any](e *Execution, name string, condition bool, then func() (T, error)) (T, error) {
	var zero T
	if !condition {
		return zero, nil
	}
	out, err := then()
	if err != nil {
		return zero, err
	}
	if e.replay != nil {
		if st := e.replay.take(name); st != nil && st.invokeState == stepStateDone {
			e.journal.restore(name, StepOptions{}, true, st)
			return out, nil
		}
	}
	rec := e.journal.begin(name, StepOptions{}, true)
	// then returning undefined leaves the output undefined
	e.journal.succeed(rec, out, !isNilOutput(any(out)), nil, false)
	return out, nil
}
