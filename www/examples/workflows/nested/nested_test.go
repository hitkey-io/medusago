package nested

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestNestedCompensatesWithParent(t *testing.T) {
	var log []string
	step := func(name string, fail bool) *composer.Step[any, any] {
		return composer.CreateStep(name, func(composer.StepExecutionContext, any) (composer.StepResponse[any], error) {
			if fail {
				return composer.StepResponse[any]{}, errors.New("fail " + name)
			}
			return composer.NewStepResponse[any](name), nil
		}, func(composer.StepExecutionContext, any) error { log = append(log, "comp "+name); return nil })
	}
	inner := composer.CreateWorkflow("inner", func(e *composer.Execution, _ any) (any, error) {
		return step("inner-step", false).Run(e, nil)
	})
	outer := composer.CreateWorkflow("outer", func(e *composer.Execution, _ any) (any, error) {
		if _, err := inner.RunAsStep(e, nil); err != nil {
			return nil, err
		}
		return step("outer-step", true).Run(e, nil)
	})
	if _, err := outer.Run(context.Background(), container{}, nil); err == nil {
		t.Fatal("expected error")
	}
	if want := []string{"comp outer-step", "comp inner-step"}; !reflect.DeepEqual(log, want) {
		t.Fatalf("%v", log)
	}
}
