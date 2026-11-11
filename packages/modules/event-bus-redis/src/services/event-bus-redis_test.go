package services

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	eventbus "github.com/hitkey-io/medusago/packages/core/utils/src/event-bus"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// Interop with the upstream RedisEventBusService on the same Redis: needs
// MEDUSA_TEST_REDIS_URL (its event bus keys are deleted) and the built reference
// checkout (MEDUSA_REFERENCE_DIR, default reference/medusa).

func setup(t *testing.T) (string, string) {
	t.Helper()
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL not set")
	}
	ref := os.Getenv("MEDUSA_REFERENCE_DIR")
	if ref == "" {
		ref, _ = filepath.Abs("../../../../../reference/medusa")
	}
	if _, err := os.Stat(filepath.Join(ref, "packages/modules/event-bus-redis/dist")); err != nil {
		t.Skip("built reference not available")
	}
	opt, _ := redis.ParseURL(url)
	c := redis.NewClient(opt)
	defer c.Close()
	deleteKeys(t, c, "RedisEventBusService:*", "staging:*")
	return url, ref
}

func node(t *testing.T, ref, url, action string, input any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(input)
	cmd := exec.Command("node", "testdata/interop.js", ref, url, action, string(b))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("node %s: %v", action, err)
	}
	var res map[string]any
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("node output %q: %v", out, err)
	}
	return res
}

func newService(t *testing.T, url string) *RedisEventBusService {
	t.Helper()
	s, err := NewRedisEventBusService(context.Background(), nil, Options{RedisURL: url})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.OnApplicationShutdown(context.Background()) })
	return s
}

func record(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// TestGoConsumesNodeEvents: events emitted by Node.js (direct and grouped
// then released) are processed by the Go worker's subscribers.
func TestGoConsumesNodeEvents(t *testing.T) {
	url, ref := setup(t)
	s := newService(t, url)
	var mu sync.Mutex
	var got []string
	done := make(chan struct{})
	sub := func(ctx context.Context, e eventbus.Event) error {
		data, _ := e.Data.(*dal.Record)
		id, _ := data.Get("id")
		keys := e.Metadata.Keys()
		sort.Strings(keys)
		_, isTime := mustGet(e.Metadata, "created_at").(time.Time)
		mu.Lock()
		defer mu.Unlock()
		got = append(got, e.Name+":"+id.(string)+":"+strings.Join(keys, ",")+":"+boolString(isTime))
		if len(got) == 2 {
			close(done)
		}
		return nil
	}
	s.Subscribe("node.direct", sub)
	s.Subscribe("node.grouped", sub)
	if err := s.OnApplicationStart(context.Background()); err != nil {
		t.Fatal(err)
	}
	node(t, ref, url, "emit", map[string]any{"release": true, "messages": []any{
		map[string]any{"name": "node.direct", "data": map[string]any{"id": "a"}},
		map[string]any{"name": "node.grouped", "data": map[string]any{"id": "b"}, "metadata": map[string]any{"eventGroupId": "g1"}},
	}})
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("timeout: %v", got)
	}
	sort.Strings(got)
	want := "node.direct:a:created_at,published_at:true,node.grouped:b:created_at,eventGroupId,published_at:true"
	if strings.Join(got, ",") != want {
		t.Fatalf("got %v", got)
	}
}

// TestNodeConsumesGoEvents: events emitted by Go (direct, grouped then
// released) are processed by the Node.js worker's subscribers.
func TestNodeConsumesGoEvents(t *testing.T) {
	url, ref := setup(t)
	s := newService(t, url)
	noop := func(context.Context, eventbus.Event) error { return nil }
	s.Subscribe("go.direct", noop)
	s.Subscribe("go.grouped", noop)
	ctx := context.Background()
	err := s.Emit(ctx, []composer.Message{
		{Name: "go.direct", Data: record("id", "c")},
		{Name: "go.grouped", Data: record("id", "d"), Metadata: record("eventGroupId", "g2")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ReleaseGroupedEvents(ctx, "g2"); err != nil {
		t.Fatal(err)
	}
	res := node(t, ref, url, "consume", map[string]any{"events": []string{"go.direct", "go.grouped"}, "count": 2})
	b, _ := json.Marshal(res["received"])
	var received []map[string]any
	_ = json.Unmarshal(b, &received)
	sort.Slice(received, func(i, j int) bool { return received[i]["name"].(string) < received[j]["name"].(string) })
	got, _ := json.Marshal(received)
	want := `[{"createdAtIsDate":true,"data":{"id":"c"},"metadataKeys":["created_at","published_at"],"name":"go.direct"},` +
		`{"createdAtIsDate":true,"data":{"id":"d"},"metadataKeys":["created_at","eventGroupId","published_at"],"name":"go.grouped"}]`
	if string(got) != want {
		t.Fatalf("got %s", got)
	}
}

// TestStagedEventsMatchNode: the staged JSON of a grouped event is the one
// Node.js writes (but the created_at value).
func TestStagedEventsMatchNode(t *testing.T) {
	url, ref := setup(t)
	res := node(t, ref, url, "stage", map[string]any{"messages": []any{
		map[string]any{"name": "x.staged", "data": map[string]any{"id": "e", "n": 1}, "metadata": map[string]any{"eventGroupId": "g3"},
			"options": map[string]any{"attempts": 3}},
	}})
	s := newService(t, url)
	ctx := context.Background()
	if err := s.Emit(ctx, []composer.Message{{Name: "x.staged", Data: record("id", "e", "n", 1), Metadata: record("eventGroupId", "g4"),
		Options: record("attempts", 3)}}); err != nil {
		t.Fatal(err)
	}
	staged, err := s.client.LRange(ctx, "staging:g4", 0, -1).Result()
	if err != nil || len(staged) != 1 {
		t.Fatalf("staged %v %v", staged, err)
	}
	norm := func(s string) string {
		s = strings.Replace(s, "g4", "g3", 1)
		i := strings.Index(s, `"created_at":"`)
		return s[:i] + s[i+len(`"created_at":"2026-01-01T00:00:00.000Z`):]
	}
	want := res["staged"].([]any)[0].(string)
	if norm(staged[0]) != norm(want) {
		t.Fatalf("staged\n%s\nwant\n%s", staged[0], want)
	}
}

func mustGet(r *dal.Record, k string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(k)
	return v
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func deleteKeys(t *testing.T, c *redis.Client, patterns ...string) {
	t.Helper()
	ctx := context.Background()
	for _, p := range patterns {
		keys, err := c.Keys(ctx, p).Result()
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) > 0 {
			c.Del(ctx, keys...)
		}
	}
}
