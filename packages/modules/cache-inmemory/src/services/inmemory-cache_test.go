package services

import (
	"context"
	"testing"
	"time"

	cachetypes "github.com/hitkey-io/medusago/packages/core/types/src/cache"
)

func TestInMemoryCache(t *testing.T) {
	ctx := context.Background()
	c := NewInMemoryCacheService(nil)
	if v, _ := c.Get(ctx, "missing"); v != nil {
		t.Fatal(v)
	}
	_ = c.Set(ctx, "ps:1", "a", nil)
	_ = c.Set(ctx, "ps:2", "b", nil)
	_ = c.Set(ctx, "other", "c", nil)
	_ = c.Set(ctx, "zero", "d", cachetypes.TTL(0))
	if v, _ := c.Get(ctx, "ps:1"); v != "a" {
		t.Fatal(v)
	}
	if v, _ := c.Get(ctx, "zero"); v != nil {
		t.Fatal("ttl 0 cached")
	}
	_ = c.Invalidate(ctx, "ps:*")
	if v, _ := c.Get(ctx, "ps:2"); v != nil {
		t.Fatal("pattern not invalidated")
	}
	if v, _ := c.Get(ctx, "other"); v != "c" {
		t.Fatal(v)
	}
	_ = c.Set(ctx, "short", "e", cachetypes.TTL(0.05))
	time.Sleep(80 * time.Millisecond)
	if v, _ := c.Get(ctx, "short"); v != nil {
		t.Fatal("expired value returned")
	}
}
