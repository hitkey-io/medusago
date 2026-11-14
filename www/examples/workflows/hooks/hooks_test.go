package hooks

import (
	"context"
	"errors"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestHookHandlerAndResult(t *testing.T) {
	ran := ""
	CreateOrderWorkflow.RegisterHook("orderCreated",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			id, _ := data.(*dal.Record).Get("order_id")
			ran = id.(string)
			return composer.NewStepResponse[any](nil), nil
		}, nil)
	if _, err := CreateOrderWorkflow.Run(context.Background(), container{}, CreateOrderInput{OrderID: "o1"}); err != nil || ran != "o1" {
		t.Fatalf("%v %q", err, ran)
	}

	PricingWorkflow.RegisterHook("setPricingContext",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			r := dal.NewRecord()
			r.Set("customer_group", "vip")
			return composer.NewStepResponse[any](r), nil
		}, nil)
	out, err := PricingWorkflow.Run(context.Background(), container{}, CreateOrderInput{OrderID: "o1"})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := out.Get("customer_group"); v != "vip" {
		t.Fatalf("%v", out)
	}
}

func TestHookCompensation(t *testing.T) {
	var got any
	wf := composer.CreateWorkflow("hook-comp", func(e *composer.Execution, _ any) (string, error) {
		if err := composer.CreateHook(e, "h", dal.NewRecord()); err != nil {
			return "", err
		}
		return "", errors.New("later failure")
	})
	wf.RegisterHook("h",
		func(ctx composer.StepExecutionContext, data any) (composer.StepResponse[any], error) {
			return composer.NewStepResponse[any](nil, []string{"id_1"}), nil
		},
		func(ctx composer.StepExecutionContext, data any) error { got = data; return nil })
	if _, err := wf.Run(context.Background(), container{}, nil); err == nil {
		t.Fatal("expected error")
	}
	if ids, ok := got.([]string); !ok || len(ids) != 1 {
		t.Fatalf("%#v", got)
	}
}
