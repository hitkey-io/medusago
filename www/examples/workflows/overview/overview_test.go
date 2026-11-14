package overview

import (
	"context"
	"testing"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

type fakeProducts struct{}

func (fakeProducts) Call(_ context.Context, method string, _ []any) (any, error) {
	return []any{[]any{}, 3}, nil
}

func TestHelloWorld(t *testing.T) {
	out, err := HelloWorldWorkflow.Run(context.Background(), container{}, WorkflowInput{Name: "John"})
	if err != nil || out.Message != "Hello John from step two!" {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestProductCount(t *testing.T) {
	out, err := ProductCountWorkflow.Run(context.Background(), container{"product": fakeProducts{}}, nil)
	if err != nil || out.Count != 3 {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestConditions(t *testing.T) {
	out, err := ConditionalWorkflow.Run(context.Background(), container{}, ConditionalInput{Name: "A", Greet: true})
	if err != nil || out != "Hello A from step two!" {
		t.Fatal(out, err)
	}
	out, err = NamedConditionWorkflow.Run(context.Background(), container{}, ConditionalInput{Name: "A", Greet: false})
	if err != nil || out != "" {
		t.Fatal(out, err)
	}
	out, err = NamedConditionWorkflow.Run(context.Background(), container{}, ConditionalInput{Name: "A", Greet: true})
	if err != nil || out != "Hello A from step two!" {
		t.Fatal(out, err)
	}
}
