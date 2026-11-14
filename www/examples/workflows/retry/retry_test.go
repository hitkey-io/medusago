package retry

import (
	"context"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestRetries(t *testing.T) {
	_, err := MyWorkflow.Run(context.Background(), container{}, nil)
	if err == nil || err.Error() != "Oops! Something happened." {
		t.Fatal(err)
	}
}

func TestRetryWaitingStep(t *testing.T) {
	done := make(chan any, 1)
	composer.Subscribe("import-job", "tx-import", "s", func(ev composer.WorkflowEvent) {
		if ev.EventType == "onFinish" {
			done <- ev.Result
		}
	})
	ctx := context.Background()
	_, _, err := ImportWorkflow.RunWithInfo(ctx, container{}, "job_1", composer.RunOptions{TransactionID: "tx-import", Notify: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := RetryWaitForImport(ctx, container{}, "tx-import"); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-done:
		if r != "import completed" {
			t.Fatalf("%v", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
