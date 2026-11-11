package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
	workflowengineredis "github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src"
	"github.com/hitkey-io/medusago/packages/modules/workflow-engine-redis/src/services"
)

// Instances (containers) of the Redis engine on the same Redis
// (MEDUSA_TEST_REDIS_URL) and database (MEDUSA_TEST_PG_ADMIN_URL): a run
// waiting for an async step stops; the response resumes it from the
// checkpoint in the instance receiving it, even after the instance that
// started it was shut down; the notifications reach the subscribers of
// the other instance; a step timeout is a job of the medusa-workflows
// queue resumed by any instance.

type testContainer struct{ engine any }

func (c *testContainer) Resolve(key string) any {
	if key == utilsmodulessdk.WORKFLOW_ENGINE {
		return c.engine
	}
	return nil
}

// invocations counts the step runs (a replay must not run a step again).
var invocations = map[string]*atomic.Int32{"before": {}, "after": {}, "compensate": {}}

type beforeOut struct {
	Value string     `json:"value"`
	Rec   *dal.Record `json:"rec"`
}

var beforeStep = composer.CreateStep("redis-test-before",
	func(_ composer.StepExecutionContext, in string) (composer.StepResponse[beforeOut], error) {
		invocations["before"].Add(1)
		r := dal.NewRecord()
		r.Set("n", 1.0)
		return composer.NewStepResponse(beforeOut{Value: in + "-before", Rec: r}, in+"-comp"), nil
	}, func(_ composer.StepExecutionContext, in any) error {
		if in == "timeout-comp" || in == "failure-comp" {
			invocations["compensate"].Add(1)
		}
		return nil
	})

var waitStep = composer.CreateStep("redis-test-wait",
	func(composer.StepExecutionContext, string) (composer.StepResponse[*dal.Record], error) {
		return composer.NewStepResponse[*dal.Record](nil), nil
	}, nil).Config("", composer.StepOptions{Async: true, Timeout: 2})

var afterStep = composer.CreateStep("redis-test-after",
	func(_ composer.StepExecutionContext, in string) (composer.StepResponse[string], error) {
		invocations["after"].Add(1)
		return composer.NewStepResponse(in), nil
	}, nil)

type asyncOut struct {
	Before beforeOut   `json:"before"`
	Wait   *dal.Record `json:"wait"`
	After  string      `json:"after"`
}

var asyncWorkflow = composer.CreateWorkflowWithOptions("redis-test-async", composer.WorkflowOptions{Store: true, RetentionTime: 3600},
	func(e *composer.Execution, in string) (asyncOut, error) {
		b, err := beforeStep.Run(e, in)
		if err != nil {
			return asyncOut{}, err
		}
		w, err := waitStep.Run(e, in)
		if err != nil {
			return asyncOut{}, err
		}
		n, _ := w.Get("n")
		a, err := afterStep.Run(e, b.Value+"|"+toString(b.Rec)+"|"+toString(n))
		if err != nil {
			return asyncOut{}, err
		}
		return asyncOut{Before: b, Wait: w, After: a}, nil
	})

func toString(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := os.Getenv("MEDUSA_TEST_PG_ADMIN_URL")
	if admin == "" {
		t.Skip("MEDUSA_TEST_PG_ADMIN_URL not set")
	}
	ctx := context.Background()
	name := os.Getenv("MEDUSA_TEST_DB_PREFIX") + "workflow_engine_redis_test"
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{"drop database if exists " + name + " with (force)", "create database " + name} {
		if _, err := conn.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	conn.Close(ctx)
	u, _ := url.Parse(admin)
	u.Path = "/" + name
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `create table "workflow_execution" ("id" varchar not null, "workflow_id" varchar not null, "transaction_id" varchar not null,
"execution" jsonb, "context" jsonb, "state" varchar not null, "created_at" timestamp default now() not null, "updated_at" timestamp default now() not null,
"deleted_at" timestamp, "retention_time" integer, "run_id" text not null, primary key ("workflow_id", "transaction_id", "run_id"))`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func newEngine(t *testing.T, url string, db *pgxpool.Pool) (*services.WorkflowsModuleService, composer.Container) {
	t.Helper()
	c := &testContainer{}
	svc, err := services.NewWorkflowsModuleService(modulessdk.ServiceOptions{Container: c, DB: db, Options: map[string]any{"redis": map[string]any{"redisUrl": url}}}, workflowengineredis.JoinerConfig)
	if err != nil {
		t.Fatal(err)
	}
	c.engine = svc
	if err := svc.OnApplicationStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdown(svc) })
	return svc, c
}

var shutdowns sync.Map

// shutdown shuts an instance down once.
func shutdown(svc *services.WorkflowsModuleService) {
	if _, done := shutdowns.LoadOrStore(svc, true); !done {
		_ = svc.OnApplicationShutdown(context.Background())
	}
}

func setup(t *testing.T) (string, *pgxpool.Pool, *redis.Client) {
	t.Helper()
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL not set")
	}
	db := testDB(t)
	opt, _ := redis.ParseURL(url)
	rc := redis.NewClient(opt)
	t.Cleanup(func() { rc.Close() })
	ctx := context.Background()
	for _, p := range []string{"dtrx:*", "bull:medusa-workflows*", "bull:workflows-cleaner*"} {
		if keys := rc.Keys(ctx, p).Val(); len(keys) > 0 {
			rc.Del(ctx, keys...)
		}
	}
	for _, c := range invocations {
		c.Store(0)
	}
	return url, db, rc
}

func executionState(t *testing.T, db *pgxpool.Pool, tx string) (string, map[string]any) {
	t.Helper()
	var state string
	var ctxRaw []byte
	err := db.QueryRow(context.Background(), `select "state", "context"::text from "workflow_execution" where "transaction_id" = $1`, tx).Scan(&state, &ctxRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var c map[string]any
	_ = json.Unmarshal(ctxRaw, &c)
	return state, c
}

func waitState(t *testing.T, db *pgxpool.Pool, tx, want string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for {
		state, c := executionState(t, db, tx)
		if state == want {
			return c
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: state %q, want %q", tx, state, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func TestStepCompletedAfterRestart(t *testing.T) {
	url, db, rc := setup(t)
	ctx := context.Background()
	a, ca := newEngine(t, url, db)

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
	if n := rc.Exists(ctx, "bull:medusa-workflows:step_timeout:redis-test-async:tx-redis-test:_root.redis-test-before.redis-test-wait").Val(); n != 1 {
		t.Fatalf("step timeout job not scheduled")
	}
	// instance a goes away; instance b receives the response
	shutdown(a)
	_, cb := newEngine(t, url, db)

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

	key := composer.IdempotencyKey{Action: "invoke", TransactionID: "tx-redis-test", StepID: "redis-test-wait", WorkflowID: "redis-test-async"}
	resp := dal.NewRecord()
	resp.Set("n", 7.0)
	if err := composer.SetStepSuccess(ctx, cb, key, resp); err != nil {
		t.Fatalf("set step success from b: %v", err)
	}
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatalf("not finished, events %v", events)
	}
	mu.Lock()
	got := strings.Join(events, ",")
	mu.Unlock()
	if got != "onResume,onStepSuccess,onStepBegin,onStepSuccess,onFinish" {
		t.Fatalf("events %s", got)
	}
	// the replay did not run the first step again; the stored output was
	// restored with its type
	if n := invocations["before"].Load(); n != 1 {
		t.Fatalf("before ran %d times", n)
	}
	c := waitState(t, db, "tx-redis-test", "done")
	data, _ := c["data"].(map[string]any)
	invoke, _ := data["invoke"].(map[string]any)
	after, _ := invoke["redis-test-after"].(map[string]any)
	out, _ := after["output"].(map[string]any)
	if out["output"] != `in-before|{"n":1}|7` {
		t.Fatalf("after output %v", after)
	}
	if rc.Exists(ctx, "dtrx:redis-test-async:tx-redis-test").Val() != 0 {
		t.Fatalf("checkpoint not removed")
	}
	if rc.Exists(ctx, "bull:medusa-workflows:step_timeout:redis-test-async:tx-redis-test:_root.redis-test-before.redis-test-wait").Val() != 0 {
		t.Fatalf("step timeout job not removed")
	}
	// a second response: the transaction is finished
	if err := composer.SetStepSuccess(ctx, cb, key, nil); err == nil || !strings.Contains(err.Error(), "could not be found") {
		t.Fatalf("finished: %v", err)
	}
	key.TransactionID = "tx-unknown"
	if err := composer.SetStepSuccess(ctx, cb, key, nil); err == nil || !strings.Contains(err.Error(), "could not be found") {
		t.Fatalf("unknown: %v", err)
	}
}

func TestStepFailureCompensatesReplayedSteps(t *testing.T) {
	url, db, _ := setup(t)
	ctx := context.Background()
	_, ca := newEngine(t, url, db)
	_, cb := newEngine(t, url, db)
	if _, _, err := asyncWorkflow.RunWithInfo(ctx, ca, "failure", composer.RunOptions{TransactionID: "tx-redis-failure"}); err != nil {
		t.Fatal(err)
	}
	key := composer.IdempotencyKey{Action: "invoke", TransactionID: "tx-redis-failure", StepID: "redis-test-wait", WorkflowID: "redis-test-async"}
	if err := composer.SetStepFailure(ctx, cb, key, errors.New("refused")); err != nil {
		t.Fatal(err)
	}
	waitState(t, db, "tx-redis-failure", "reverted")
	if n := invocations["compensate"].Load(); n != 1 {
		t.Fatalf("compensated %d times", n)
	}
	// a concurrent second response sees the step finished
	if err := composer.SetStepFailure(ctx, ca, key, errors.New("again")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestStepTimeoutJobResumedByAnotherInstance(t *testing.T) {
	url, db, _ := setup(t)
	ctx := context.Background()
	a, ca := newEngine(t, url, db)
	if _, _, err := asyncWorkflow.RunWithInfo(ctx, ca, "timeout", composer.RunOptions{TransactionID: "tx-redis-timeout"}); err != nil {
		t.Fatal(err)
	}
	shutdown(a)
	newEngine(t, url, db)
	c := waitState(t, db, "tx-redis-timeout", "reverted")
	if n := invocations["compensate"].Load(); n != 1 {
		t.Fatalf("compensated %d times", n)
	}
	errs, _ := c["errors"].([]any)
	if len(errs) != 1 || !strings.Contains(toString(errs[0]), `"action":"redis-test-wait"`) {
		t.Fatalf("errors %v", errs)
	}
}
