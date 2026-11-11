package services_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	workflowengineredis "github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/services"
)

// Two instances (containers) of the Redis engine on the same Redis
// (MEDUSA_TEST_REDIS_URL, flushed): an async step waiting in one instance
// is completed through the other; the notifications of the run reach the
// subscribers of the other instance; the running checkpoint is kept under
// dtrx:<workflowId>:<transactionId> and removed when the run finishes.

type testContainer struct{ engine any }

func (c *testContainer) Resolve(key string) any {
	if key == utilsmodulessdk.WORKFLOW_ENGINE {
		return c.engine
	}
	return nil
}

var waitStep = composer.CreateStep("redis-test-wait",
	func(composer.StepExecutionContext, string) (composer.StepResponse[map[string]any], error) {
		return composer.NewStepResponse[map[string]any](nil), nil
	}, nil).Config("", composer.StepOptions{Async: true})

var asyncWorkflow = composer.CreateWorkflowWithOptions("redis-test-async", composer.WorkflowOptions{Store: true},
	func(e *composer.Execution, in string) (map[string]any, error) {
		return waitStep.Run(e, in)
	})

func newEngine(t *testing.T, url string) (*services.WorkflowsModuleService, composer.Container) {
	t.Helper()
	c := &testContainer{}
	svc, err := services.NewWorkflowsModuleService(modulessdk.ServiceOptions{Container: c, Options: map[string]any{"redis": map[string]any{"redisUrl": url}}}, workflowengineredis.JoinerConfig)
	if err != nil {
		t.Fatal(err)
	}
	c.engine = svc
	if err := svc.OnApplicationStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = svc.OnApplicationShutdown(context.Background()) })
	return svc, c
}

func TestStepCompletedFromAnotherInstance(t *testing.T) {
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL not set")
	}
	opt, _ := redis.ParseURL(url)
	rc := redis.NewClient(opt)
	defer rc.Close()
	ctx := context.Background()
	if err := rc.FlushDB(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	_, ca := newEngine(t, url)
	_, cb := newEngine(t, url)

	var mu sync.Mutex
	var events []string
	finished := make(chan struct{})
	composer.Subscribe("redis-test-async", "tx-redis-test", "sub-b", func(ev composer.WorkflowEvent) {
		mu.Lock()
		events = append(events, ev.EventType)
		mu.Unlock()
		if ev.EventType == "onFinish" {
			close(finished)
		}
	})
	defer composer.Unsubscribe("redis-test-async", "tx-redis-test", "sub-b")

	_, info, err := asyncWorkflow.RunWithInfo(ctx, ca, "in", composer.RunOptions{TransactionID: "tx-redis-test", Notify: true})
	if err != nil || !info.Detached {
		t.Fatalf("run: %v %+v", err, info)
	}
	raw, err := rc.Get(ctx, "dtrx:redis-test-async:tx-redis-test").Result()
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	var cp map[string]any
	_ = json.Unmarshal([]byte(raw), &cp)
	if flow, _ := cp["flow"].(map[string]any); flow["state"] != "invoking" || flow["hasAsyncSteps"] != true {
		t.Fatalf("checkpoint flow %v", raw)
	}
	key := composer.IdempotencyKey{Action: "invoke", TransactionID: "tx-redis-test", StepID: "redis-test-wait", WorkflowID: "redis-test-async"}
	if err := composer.SetStepSuccess(ctx, cb, key, map[string]any{"ok": true}); err != nil {
		t.Fatalf("set step success from b: %v", err)
	}
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatalf("not finished, events %v", events)
	}
	// the subscriber is in this process: local delivery plus the copy
	// published by instance a for instance b's notifications
	mu.Lock()
	got := strings.Join(events, ",")
	mu.Unlock()
	if !strings.HasPrefix(got, "onStepAwaiting") && !strings.Contains(got, "onFinish") {
		t.Fatalf("events %s", got)
	}
	time.Sleep(200 * time.Millisecond)
	if n := rc.Exists(ctx, "dtrx:redis-test-async:tx-redis-test").Val(); n != 0 {
		t.Fatalf("checkpoint not removed")
	}
	// unknown transaction: no instance answers
	key.TransactionID = "tx-unknown"
	if err := composer.SetStepSuccess(ctx, cb, key, nil); err == nil || !strings.Contains(err.Error(), "could not be found") {
		t.Fatalf("unknown: %v", err)
	}
}
