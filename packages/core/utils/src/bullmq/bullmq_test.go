package bullmq

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// The interop tests need a Redis server (MEDUSA_TEST_REDIS_URL) and the
// upstream checkout with node_modules (MEDUSA_REFERENCE_DIR, default
// reference/medusa at the repository root) to run bullmq itself.

func testRedis(t *testing.T) (*redis.Client, string) {
	t.Helper()
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL not set")
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		t.Fatal(err)
	}
	c := redis.NewClient(opt)
	t.Cleanup(func() { _ = c.Close() })
	return c, url
}

func referenceDir(t *testing.T) string {
	t.Helper()
	dir := os.Getenv("MEDUSA_REFERENCE_DIR")
	if dir == "" {
		dir, _ = filepath.Abs("../../../../../reference/medusa")
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules/bullmq")); err != nil {
		t.Skip("bullmq reference not available")
	}
	return dir
}

func nodeRun(t *testing.T, url, action string, input any) map[string]any {
	t.Helper()
	ref := referenceDir(t)
	b, _ := json.Marshal(input)
	cmd := exec.Command("node", "testdata/interop.js", ref, url, action, string(b))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node %s: %v", action, err)
	}
	var res map[string]any
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("node %s output %q: %v", action, out, err)
	}
	return res
}

func cleanPrefix(t *testing.T, c *redis.Client, prefix string) {
	t.Helper()
	ctx := context.Background()
	keys, _ := c.Keys(ctx, prefix+":*").Result()
	if len(keys) > 0 {
		c.Del(ctx, keys...)
	}
	t.Cleanup(func() {
		keys, _ := c.Keys(ctx, prefix+":*").Result()
		if len(keys) > 0 {
			c.Del(ctx, keys...)
		}
	})
}

// TestAddMatchesNode adds the same jobs from Node.js and from Go: the job
// hashes (but the timestamps) and the queue structures are identical.
func TestAddMatchesNode(t *testing.T) {
	c, url := testRedis(t)
	ctx := context.Background()
	jobs := []map[string]any{
		{"name": "standard", "data": map[string]any{"data": map[string]any{"id": "x"}, "metadata": map[string]any{"eventGroupId": "g"}},
			"opts": map[string]any{"removeOnComplete": true, "attempts": 3, "backoff": 1000}},
		{"name": "prioritized", "data": map[string]any{"a": 1}, "opts": map[string]any{"removeOnComplete": true, "attempts": 1, "priority": 100}},
		{"name": "delayed", "data": map[string]any{"a": []any{1, "b"}}, "opts": map[string]any{"delay": 60000, "removeOnComplete": map[string]any{"age": 10}}},
		{"name": "custom", "data": map[string]any{}, "opts": map[string]any{"jobId": "custom-id", "removeOnFail": 5}},
	}
	snapshot := func(prefix string) map[string]any {
		out := map[string]any{}
		keys, _ := c.Keys(ctx, prefix+":*").Result()
		for _, k := range keys {
			short := strings.TrimPrefix(k, prefix+":")
			switch c.Type(ctx, k).Val() {
			case "hash":
				h := c.HGetAll(ctx, k).Val()
				delete(h, "timestamp")
				out[short] = h
			case "list":
				out[short] = c.LRange(ctx, k, 0, -1).Val()
			case "zset":
				z := c.ZRangeWithScores(ctx, k, 0, -1).Val()
				var members []string
				for _, e := range z {
					members = append(members, e.Member.(string))
				}
				out[short] = members
			case "string":
				out[short] = c.Get(ctx, k).Val()
			case "stream":
				n := c.XLen(ctx, k).Val()
				out[short] = n
			}
		}
		return out
	}
	cleanPrefix(t, c, "node-add")
	cleanPrefix(t, c, "go-add")
	nodeRun(t, url, "add", map[string]any{"queue": "q", "prefix": "node-add", "jobs": jobs})
	q := NewQueue(ctx, "q", c, QueueOptions{Prefix: "go-add"})
	var bulk []BulkJob
	for _, j := range jobs {
		bulk = append(bulk, BulkJob{Name: j["name"].(string), Data: ObjFromMap(j["data"]), Opts: ObjFromMap(j["opts"])})
	}
	if _, err := q.AddBulk(ctx, bulk); err != nil {
		t.Fatal(err)
	}
	want, got := snapshot("node-add"), snapshot("go-add")
	// delayed scores embed the timestamp
	wb, _ := json.MarshalIndent(want, "", " ")
	gb, _ := json.MarshalIndent(got, "", " ")
	if string(wb) != string(gb) {
		t.Fatalf("redis state differs\n--- node\n%s\n--- go\n%s", wb, gb)
	}
}

// TestNodeConsumesGoJobs: jobs added by Go are processed by a Node.js
// worker (with a retry), and the reverse.
func TestNodeConsumesGoJobs(t *testing.T) {
	c, url := testRedis(t)
	ctx := context.Background()
	cleanPrefix(t, c, "interop-a")
	q := NewQueue(ctx, "events", c, QueueOptions{Prefix: "interop-a"})
	if _, err := q.Add(ctx, "order.placed", NewObj("data", NewObj("id", "order_1")), NewObj("attempts", 2, "removeOnComplete", true)); err != nil {
		t.Fatal(err)
	}
	res := nodeRun(t, url, "consume", map[string]any{"queue": "events", "prefix": "interop-a", "count": 2, "fail": true, "updateData": true})
	processed := res["processed"].([]any)
	if len(processed) != 2 {
		t.Fatalf("processed %v", processed)
	}
	second := processed[1].(map[string]any)
	if second["attemptsMade"].(float64) != 1 {
		t.Fatalf("retry attemptsMade %v", second)
	}
	if ids := second["data"].(map[string]any)["completedSubscriberIds"]; ids == nil {
		t.Fatalf("updateData lost: %v", second)
	}
	counts, _ := q.GetCounts(ctx, "completed", "failed", "wait", "active")
	if counts["completed"] != 0 || counts["failed"] != 0 || counts["wait"] != 0 || counts["active"] != 0 {
		t.Fatalf("counts %v", counts)
	}
}

func TestGoConsumesNodeJobs(t *testing.T) {
	c, url := testRedis(t)
	ctx := context.Background()
	cleanPrefix(t, c, "interop-b")
	nodeRun(t, url, "add", map[string]any{"queue": "events", "prefix": "interop-b", "jobs": []map[string]any{
		{"name": "product.created", "data": map[string]any{"data": map[string]any{"id": "prod_1"}}, "opts": map[string]any{"attempts": 3, "backoff": map[string]any{"type": "fixed", "delay": 100}, "removeOnComplete": true}},
		{"name": "later", "data": map[string]any{"n": 2}, "opts": map[string]any{"delay": 300, "removeOnComplete": true}},
		{"name": "prio", "data": map[string]any{"n": 3}, "opts": map[string]any{"priority": 10, "removeOnComplete": true}},
	}})
	var mu sync.Mutex
	var seen []string
	done := make(chan struct{})
	w := NewWorker("events", c, func(ctx context.Context, j *Job) (any, error) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, j.Name+"#"+j.AttemptsMadeOrZero())
		if j.Name == "product.created" && j.AttemptsMade == 0 {
			if err := j.UpdateData(ctx, ObjFromMap(j.Data).Clone().Set("completedSubscriberIds", []any{"a"})); err != nil {
				return nil, err
			}
			return nil, errors.New("subscriber failed")
		}
		if len(seen) == 4 {
			close(done)
		}
		return map[string]any{"ok": true}, nil
	}, WorkerOptions{Prefix: "interop-b", DrainDelay: 0.2})
	w.Run(ctx)
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout, seen %v", seen)
	}
	w.Close()
	sort.Strings(seen)
	if strings.Join(seen, ",") != "later#0,prio#0,product.created#0,product.created#1" {
		t.Fatalf("seen %v", seen)
	}
	counts, _ := q(c).GetCounts(ctx, "completed", "failed", "wait", "delayed", "active")
	for k, v := range counts {
		if v != 0 {
			t.Fatalf("%s: %d", k, v)
		}
	}
}

func q(c *redis.Client) *Queue { return newQueueBase("events", c, QueueOptions{Prefix: "interop-b"}) }

// TestRepeatMatchesNode: the repeatable job structures written by Go and
// Node.js for the same repeat options are identical.
func TestRepeatMatchesNode(t *testing.T) {
	c, url := testRedis(t)
	ctx := context.Background()
	cleanPrefix(t, c, "rep-node")
	cleanPrefix(t, c, "rep-go")
	opts := map[string]any{"repeat": map[string]any{"every": 3600000, "key": "schedule_job-test"},
		"removeOnComplete": map[string]any{"age": 86400, "count": 1000}, "removeOnFail": map[string]any{"age": 604800, "count": 5000}}
	data := map[string]any{"jobId": "job-test", "schedulerOptions": map[string]any{"interval": 3600000, "concurrency": "forbid"}}
	res := nodeRun(t, url, "repeat", map[string]any{"queue": "jobs", "prefix": "rep-node", "name": "schedule", "data": data, "opts": opts})
	gq := NewQueue(ctx, "jobs", c, QueueOptions{Prefix: "rep-go"})
	job, err := gq.Add(ctx, "schedule", ObjFromMap(data), ObjFromMap(opts))
	if err != nil {
		t.Fatal(err)
	}
	if job.ID != res["id"] {
		t.Fatalf("job id %s != %v", job.ID, res["id"])
	}
	nh := c.HGetAll(ctx, "rep-node:jobs:"+job.ID).Val()
	gh := c.HGetAll(ctx, "rep-go:jobs:"+job.ID).Val()
	for _, h := range []map[string]string{nh, gh} {
		delete(h, "timestamp")
		delete(h, "delay")
	}
	nb, _ := json.Marshal(nh)
	gb, _ := json.Marshal(gh)
	// opts embeds the "timestamp" of creation too
	norm := func(s string) string {
		var m map[string]any
		_ = json.Unmarshal([]byte(s), &m)
		if o, ok := m["opts"].(string); ok {
			var om map[string]any
			_ = json.Unmarshal([]byte(o), &om)
			delete(om, "timestamp")
			delete(om, "delay")
			b, _ := json.Marshal(om)
			m["opts"] = string(b)
		}
		b, _ := json.Marshal(m)
		return string(b)
	}
	if norm(string(nb)) != norm(string(gb)) {
		t.Fatalf("repeat job differs\n%s\n%s", nb, gb)
	}
	rn, _ := newQueueBase("jobs", c, QueueOptions{Prefix: "rep-node"}).GetRepeatableJobs(ctx)
	rg, _ := gq.GetRepeatableJobs(ctx)
	if len(rn) != 1 || len(rg) != 1 || rn[0] != rg[0] {
		t.Fatalf("repeatable %v %v", rn, rg)
	}
	if ok, err := gq.RemoveRepeatableByKey(ctx, "schedule_job-test"); err != nil || !ok {
		t.Fatalf("remove %v %v", ok, err)
	}
	if n := c.ZCard(ctx, "rep-go:jobs:delayed").Val(); n != 0 {
		t.Fatalf("delayed left %d", n)
	}
}

func (j *Job) AttemptsMadeOrZero() string {
	b, _ := json.Marshal(j.AttemptsMade)
	return string(b)
}
