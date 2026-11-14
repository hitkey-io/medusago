package errorhandling

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestWorkflowErrors(t *testing.T) {
	ctx := context.Background()
	_, err := MyWorkflow.Run(ctx, container{}, nil)
	var me *common.MedusaError
	if !stderrors.As(err, &me) || me.Message != "Oops! Something happened." {
		t.Fatalf("%v", err)
	}
	out, err := SkipWorkflow.Run(ctx, container{}, nil)
	if err != nil || out != "skipped" {
		t.Fatalf("%v %v", out, err)
	}
	out, err = ContinueWorkflow.Run(ctx, container{}, nil)
	if err != nil || out != "more actions done" {
		t.Fatalf("%v %v", out, err)
	}
}

func TestPermanentFailure(t *testing.T) {
	calls := 0
	step := composer.CreateStep("perm", func(composer.StepExecutionContext, any) (composer.StepResponse[string], error) {
		calls++
		return composer.PermanentFailure[string]("rejected"), nil
	}, nil).Config("", composer.StepOptions{MaxRetries: 3})
	wf := composer.CreateWorkflow("perm-wf", func(e *composer.Execution, _ any) (string, error) { return step.Run(e, nil) })
	_, err := wf.Run(context.Background(), container{}, nil)
	if err == nil || err.Error() != "rejected" || calls != 1 {
		t.Fatalf("%v %d", err, calls)
	}
	_ = permanentFailureStep
}
