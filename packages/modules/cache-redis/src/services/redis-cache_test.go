package services

import (
	"context"
	"os"
	"testing"
	"time"

	cachetypes "github.com/hitkey-io/medusago/packages/core/types/src/cache"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
)

func TestRedisCache(t *testing.T) {
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL is not set (e.g. redis://localhost:6379)")
	}
	ctx := context.Background()
	client := ioredis.New(url, nil)
	defer client.Close()
	ns := "medusa_cache_test_" + time.Now().Format("150405000000")
	c := NewRedisCacheService(client, nil, ns)

	// the value is JSON.stringify(data) under <namespace>:<key>, EX ttl
	r := dal.NewRecord()
	r.Set("id", "authmfachal_01")
	r.Set("methods", []any{"totp"})
	r.Set("expires_at", time.Date(2026, 11, 11, 10, 0, 0, 123456789, time.UTC))
	r.Set("attempts", float64(0))
	r.Set("completed_at", nil)
	if err := c.Set(ctx, "auth:mfa:challenge:authmfachal_01", r, cachetypes.TTL(300)); err != nil {
		t.Fatal(err)
	}
	raw, err := client.Get(ctx, ns+":auth:mfa:challenge:authmfachal_01").Result()
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"authmfachal_01","methods":["totp"],"expires_at":"2026-11-11T10:00:00.123Z","attempts":0,"completed_at":null}`
	if raw != want {
		t.Fatalf("got %s", raw)
	}
	if ttl, _ := client.TTL(ctx, ns+":auth:mfa:challenge:authmfachal_01").Result(); ttl <= 290*time.Second || ttl > 300*time.Second {
		t.Fatalf("ttl %v", ttl)
	}
	got, err := c.Get(ctx, "auth:mfa:challenge:authmfachal_01")
	if err != nil {
		t.Fatal(err)
	}
	gr, _ := got.(*dal.Record)
	if gr == nil {
		t.Fatalf("got %#v", got)
	}
	if b, _ := gr.MarshalJSON(); string(b) != want {
		t.Fatalf("round trip %s", b)
	}
	// an invalid value is unlinked
	_ = client.Set(ctx, ns+":bad", "{", 0).Err()
	if v, _ := c.Get(ctx, "bad"); v != nil {
		t.Fatal(v)
	}
	if n, _ := client.Exists(ctx, ns+":bad").Result(); n != 0 {
		t.Fatal("invalid value kept")
	}
	_ = c.Set(ctx, "ps:1", "a", nil)
	_ = c.Set(ctx, "ps:2", "b", nil)
	if err := c.Invalidate(ctx, "ps:*"); err != nil {
		t.Fatal(err)
	}
	if v, _ := c.Get(ctx, "ps:1"); v != nil {
		t.Fatal("not invalidated")
	}
	_ = c.Invalidate(ctx, "*")
}
