package services_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/types"
	"github.com/hitkey-io/medusago/packages/modules/providers/caching-redis/src/services"
)

// The Redis provider's storage format and tag invalidation (upstream
// redis-cache.ts): entries are hashes { data, options } where data is a
// format byte (0 plain, 1 gzip) followed by the JSON; tag indexes are sets
// of hashed tags; clear with options only removes the entries whose
// autoInvalidate is not false.
func TestRedisCachingProvider(t *testing.T) {
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL not set")
	}
	opt, _ := redis.ParseURL(url)
	rc := redis.NewClient(opt)
	defer rc.Close()
	ctx := context.Background()
	prefix := "mctest:"
	if keys := rc.Keys(ctx, prefix+"*").Val(); len(keys) > 0 {
		rc.Del(ctx, keys...)
	}
	threshold := 16
	p := services.NewRedisCachingProvider(rc, prefix, loaders.Hasher, nil, services.Options{CompressionThreshold: &threshold})

	small := dal.NewRecord()
	small.Set("a", 1.0)
	big := dal.NewRecord()
	big.Set("text", strings.Repeat("x", 100))
	f := false
	if err := p.Set(ctx, types.ProviderSetOptions{Key: "small", Data: small, Tags: []string{"Product:p1", "Product:list:*"}, HasOptions: true}); err != nil {
		t.Fatal(err)
	}
	if err := p.Set(ctx, types.ProviderSetOptions{Key: "big", Data: big, Tags: []string{"Product:list:*"}, HasOptions: true, AutoInvalidate: &f}); err != nil {
		t.Fatal(err)
	}
	raw, _ := rc.HGet(ctx, prefix+"small", "data").Bytes()
	if raw[0] != 0 || string(raw[1:]) != `{"a":1}` {
		t.Fatalf("small data %q", raw)
	}
	if raw, _ := rc.HGet(ctx, prefix+"big", "data").Bytes(); raw[0] != 1 {
		t.Fatalf("big data not compressed")
	}
	if o := rc.HGet(ctx, prefix+"big", "options").Val(); o != `{"autoInvalidate":false}` {
		t.Fatalf("options %q", o)
	}
	if o := rc.HGet(ctx, prefix+"small", "options").Val(); o != `{}` {
		t.Fatalf("options %q", o)
	}
	if members := rc.SMembers(ctx, prefix+"tag:"+loaders.Hasher("Product:list:*")).Val(); len(members) != 2 {
		t.Fatalf("tag index %v", members)
	}
	got, err := p.Get(ctx, "big", nil)
	if err != nil || got.(*dal.Record).Keys()[0] != "text" {
		t.Fatalf("get big: %v %v", got, err)
	}
	byTag, _ := p.Get(ctx, "", []string{"Product:p1"})
	if l, _ := byTag.([]any); len(l) != 1 {
		t.Fatalf("by tag %v", byTag)
	}
	// strategy invalidation: autoInvalidate false is kept
	tr := true
	if err := p.Clear(ctx, types.ProviderClearOptions{Tags: []string{"Product:list:*"}, Options: true, AutoInvalidate: &tr}); err != nil {
		t.Fatal(err)
	}
	if v, _ := p.Get(ctx, "small", nil); v != nil {
		t.Fatalf("small not cleared")
	}
	if v, _ := p.Get(ctx, "big", nil); v == nil {
		t.Fatalf("big cleared")
	}
	if rc.Exists(ctx, prefix+"tags:small").Val() != 0 {
		t.Fatalf("reverse index of small kept")
	}
	// explicit clear without options removes everything matching
	if err := p.Clear(ctx, types.ProviderClearOptions{Tags: []string{"Product:list:*"}}); err != nil {
		t.Fatal(err)
	}
	if v, _ := p.Get(ctx, "big", nil); v != nil {
		t.Fatalf("big not cleared")
	}
	if n := len(rc.Keys(ctx, prefix+"*").Val()); n != 0 {
		t.Fatalf("%d keys left", n)
	}
}
