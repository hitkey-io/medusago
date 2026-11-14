package longrunning

import (
	"context"
	"testing"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

type container map[string]any

func (c container) Resolve(key string) any { return c[key] }

func TestLongRunning(t *testing.T) {
	ctx := context.Background()
	finished := make(chan any, 1)
	txID, err := Start(ctx, container{}, func(result any) { finished <- result })
	if err != nil || txID == "" {
		t.Fatal(err)
	}
	if err := SetStep2Success(ctx, container{}, txID); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-finished:
		out, ok := r.(*dal.Record)
		if !ok {
			t.Fatalf("%#v", r)
		}
		if m, _ := out.Get("message"); m != "Finished three steps" {
			t.Fatalf("%#v", r)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestLongRunningFailure(t *testing.T) {
	ctx := context.Background()
	finished := make(chan any, 1)
	txID, err := Start(ctx, container{}, func(result any) { finished <- result })
	if err != nil {
		t.Fatal(err)
	}
	_ = SetStep2Failure(ctx, container{}, txID)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestBackgroundExecution(t *testing.T) {
	ctx := context.Background()
	finished := make(chan any, 1)
	composer.Subscribe("import-data", "tx-bg", "s", func(ev composer.WorkflowEvent) {
		if ev.EventType == "onFinish" {
			finished <- ev.Result
		}
	})
	_, info, err := ImportWorkflow.RunWithInfo(ctx, container{}, nil, composer.RunOptions{TransactionID: "tx-bg", Notify: true})
	if err != nil || !info.Detached {
		t.Fatalf("%+v %v", info, err)
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
