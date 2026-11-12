package composer

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryStepInvokesTheWaitingStepAgain(t *testing.T) {
	var calls atomic.Int32
	wait := CreateStep("retry-wait", func(_ StepExecutionContext, _ any) (StepResponse[any], error) {
		if calls.Add(1) == 1 {
			// the first invocation waits for a response
			return StepResponse[any]{}, nil
		}
		return NewStepResponse[any]("second"), nil
	}, nil).Config("", StepOptions{Async: true})
	got := make(chan string, 1)
	after := CreateStep("retry-after", func(_ StepExecutionContext, in any) (StepResponse[any], error) {
		got <- in.(string)
		return StepResponse[any]{}, nil
	}, nil)
	wf := CreateWorkflow("test-retry-step", func(e *Execution, _ any) (any, error) {
		out, err := wait.Run(e, nil)
		if err != nil {
			return nil, err
		}
		return after.Run(e, out)
	})
	ctx := context.Background()
	c := fakeContainer{}
	_, info, err := wf.RunWithInfo(ctx, c, nil, RunOptions{TransactionID: "tx-retry"})
	if err != nil || !info.Detached {
		t.Fatalf("run = %+v, %v", info, err)
	}
	cp, err := GetRunningTransaction(ctx, c, "test-retry-step", "tx-retry")
	if err != nil || flowState(cp) != TransactionStateInvoking {
		t.Fatalf("running transaction = %v, %v", cp, err)
	}
	if _, err := RetryStep(ctx, c, IdempotencyKey{Action: "invoke", TransactionID: "tx-retry", StepID: "retry-after", WorkflowID: "test-retry-step"}); err == nil {
		t.Fatal("expected an error retrying a step that is not waiting")
	}
	if _, err := RetryStep(ctx, c, IdempotencyKey{Action: "invoke", TransactionID: "tx-retry", StepID: "retry-wait", WorkflowID: "test-retry-step"}); err != nil {
		t.Fatal(err)
	}
	select {
	case v := <-got:
		if v != "second" || calls.Load() != 2 {
			t.Fatalf("after got %q, calls %d", v, calls.Load())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the retried step did not complete")
	}
}

func TestCancelCompensatesOnceTheWaitingStepResponds(t *testing.T) {
	var compensated []string
	first := CreateStep("cancel-first", func(_ StepExecutionContext, _ any) (StepResponse[string], error) {
		return NewStepResponse("first"), nil
	}, func(_ StepExecutionContext, in any) error {
		compensated = append(compensated, "first")
		return nil
	})
	wait := CreateStep("cancel-wait", func(_ StepExecutionContext, _ any) (StepResponse[any], error) {
		return StepResponse[any]{}, nil
	}, func(_ StepExecutionContext, in any) error {
		compensated = append(compensated, "wait")
		return nil
	}).Config("", StepOptions{Async: true})
	ranAfter := false
	after := CreateStep("cancel-after", func(_ StepExecutionContext, _ any) (StepResponse[any], error) {
		ranAfter = true
		return StepResponse[any]{}, nil
	}, nil)
	wf := CreateWorkflow("test-cancel-waiting", func(e *Execution, _ any) (any, error) {
		if _, err := first.Run(e, nil); err != nil {
			return nil, err
		}
		if _, err := wait.Run(e, nil); err != nil {
			return nil, err
		}
		return after.Run(e, nil)
	})
	ctx := context.Background()
	c := fakeContainer{}
	if _, _, err := wf.RunWithInfo(ctx, c, nil, RunOptions{TransactionID: "tx-cancel"}); err != nil {
		t.Fatal(err)
	}
	res, err := Cancel(ctx, c, "test-cancel-waiting", CancelOptions{TransactionID: "tx-cancel"})
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := res.Acknowledgement.Get("hasFinished"); v != false || flowState(res.Transaction) != TransactionStateWaitingToCompensate {
		t.Fatalf("cancel = %v, state %s", res.Acknowledgement, flowState(res.Transaction))
	}
	if _, err := Cancel(ctx, c, "test-cancel-waiting", CancelOptions{TransactionID: "tx-cancel"}); err == nil {
		t.Fatal("a transaction already compensating was cancelled again")
	}
	err = SetStepSuccess(ctx, c, IdempotencyKey{Action: "invoke", TransactionID: "tx-cancel", StepID: "cancel-wait", WorkflowID: "test-cancel-waiting"}, "ok")
	if err == nil || err.Error() != "Unknown error" {
		t.Fatalf("setStepSuccess err = %v", err)
	}
	if ranAfter || len(compensated) != 2 || compensated[0] != "wait" || compensated[1] != "first" {
		t.Fatalf("after ran %v, compensated %v", ranAfter, compensated)
	}
	if _, err := GetRunningTransaction(ctx, c, "test-cancel-waiting", "tx-cancel"); err == nil {
		t.Fatal("a reverted transaction is still running")
	}
}
