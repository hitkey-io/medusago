package parallel

import (
	"context"
	"testing"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestParallel(t *testing.T) {
	out, err := MyWorkflow.Run(context.Background(), container{}, WorkflowInput{Title: "Shirt"})
	if err != nil || out.ID != "prod_1" {
		t.Fatalf("%v %v", out, err)
	}
}
