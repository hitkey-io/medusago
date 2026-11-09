package composer

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type fakeContainer map[string]any

func (f fakeContainer) Resolve(k string) any { return f[k] }

type fakeBus struct{ released, cleared []string }

func (b *fakeBus) Emit(context.Context, []Message) error { return nil }
func (b *fakeBus) ReleaseGroupedEvents(_ context.Context, id string) error {
	b.released = append(b.released, id)
	return nil
}
func (b *fakeBus) ClearGroupedEvents(_ context.Context, id string, _ []string) error {
	b.cleared = append(b.cleared, id)
	return nil
}

func TestCompensationOrderAndFirstError(t *testing.T) {
	var log []string
	mk := func(name string, fail bool) *Step[int, int] {
		return CreateStep(name, func(_ StepExecutionContext, in int) (StepResponse[int], error) {
			log = append(log, "invoke "+name)
			if fail {
				return StepResponse[int]{}, errors.New("boom " + name)
			}
			return NewStepResponse(in+1, name+"-comp"), nil
		}, func(_ StepExecutionContext, c any) error {
			log = append(log, fmt.Sprintf("compensate %s %v", name, c))
			return nil
		})
	}
	a, b, c, d := mk("a", false), mk("b", false), mk("c", true), mk("d", false)
	wf := CreateWorkflow("wf", func(e *Execution, in int) (int, error) {
		x, err := a.Run(e, in)
		if err != nil {
			return 0, err
		}
		if x, err = b.Run(e, x); err != nil {
			return 0, err
		}
		if x, err = c.Run(e, x); err != nil {
			return 0, err
		}
		return d.Run(e, x)
	})
	bus := &fakeBus{}
	_, err := wf.Run(context.Background(), fakeContainer{"event_bus": bus}, 1)
	if err == nil || err.Error() != "boom c" {
		t.Fatalf("error: %v", err)
	}
	want := []string{"invoke a", "invoke b", "invoke c", "compensate c <nil>", "compensate b b-comp", "compensate a a-comp"}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("got %v", log)
	}
	if len(bus.cleared) != 1 || len(bus.released) != 0 {
		t.Fatalf("events: %+v", bus)
	}
}

func TestRetriesAndContinue(t *testing.T) {
	n := 0
	flaky := CreateStep("flaky", func(_ StepExecutionContext, _ any) (StepResponse[int], error) {
		n++
		if n < 3 {
			return StepResponse[int]{}, errors.New("temporary")
		}
		return NewStepResponse(n), nil
	}, nil).Config("", StepOptions{MaxRetries: 2})
	broken := CreateStep("broken", func(_ StepExecutionContext, _ any) (StepResponse[int], error) {
		return StepResponse[int]{}, errors.New("x")
	}, nil).Config("", StepOptions{ContinueOnPermanentFailure: true})
	wf := CreateWorkflow("wf2", func(e *Execution, _ any) (int, error) {
		v, err := flaky.Run(e, nil)
		if err != nil {
			return 0, err
		}
		if _, err := broken.Run(e, nil); err != nil {
			return 0, err
		}
		return v, nil
	})
	bus := &fakeBus{}
	v, err := wf.Run(context.Background(), fakeContainer{"event_bus": bus}, nil)
	if err != nil || v != 3 {
		t.Fatalf("v=%d err=%v", v, err)
	}
	if len(bus.released) != 1 {
		t.Fatal("events not released")
	}
}

func TestNestedWorkflowCompensatesAsUnit(t *testing.T) {
	var log []string
	step := func(name string, fail bool) *Step[any, any] {
		return CreateStep(name, func(_ StepExecutionContext, _ any) (StepResponse[any], error) {
			if fail {
				return StepResponse[any]{}, errors.New("fail " + name)
			}
			return NewStepResponse[any](name), nil
		}, func(_ StepExecutionContext, c any) error {
			log = append(log, fmt.Sprint("comp ", c))
			return nil
		})
	}
	inner := CreateWorkflow("inner", func(e *Execution, _ any) (any, error) {
		if _, err := step("i1", false).Run(e, nil); err != nil {
			return nil, err
		}
		return step("i2", false).Run(e, nil)
	})
	outer := CreateWorkflow("outer", func(e *Execution, _ any) (any, error) {
		if _, err := step("o1", false).Run(e, nil); err != nil {
			return nil, err
		}
		if _, err := inner.RunAsStep(e, nil); err != nil {
			return nil, err
		}
		return step("o2", true).Run(e, nil)
	})
	_, err := outer.Run(context.Background(), fakeContainer{}, nil)
	if err == nil || err.Error() != "fail o2" {
		t.Fatal(err)
	}
	want := []string{"comp <nil>", "comp i2", "comp i1", "comp o1"}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("got %v", log)
	}
}
