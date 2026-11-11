package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"os"
	"sync/atomic"
	"testing"
	"time"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/testdata"
)

// Key names and Lua commands equal upstream's, so Node and Go instances
// share the locks of one Redis.
func TestKeysAndScriptsMatchUpstream(t *testing.T) {
	v, err := testdata.Load()
	if err != nil {
		t.Fatal(err)
	}
	client := ioredis.New("redis://localhost:1", nil)
	defer client.Close()
	def := NewRedisLockingProvider(client, "medusa_lock:", nil, Options{})
	custom := NewRedisLockingProvider(client, "custom:", nil, Options{})
	for _, k := range v.Keys {
		if got := def.getKeyName(k.Key); got != k.RedisKey {
			t.Errorf("%q: got %q, want %q", k.Key, got, k.RedisKey)
		}
		if got := custom.getKeyName(k.Key); got != k.RedisKeyCustomPrefix {
			t.Errorf("%q: got %q, want %q", k.Key, got, k.RedisKeyCustomPrefix)
		}
	}
	for name, script := range map[string]interface{ Hash() string }{"acquireLock": acquireLockScript, "extendLock": extendLockScript, "releaseLock": releaseLockScript} {
		up, ok := v.Scripts[name]
		if !ok {
			t.Fatalf("missing upstream script %s", name)
		}
		if want := sha1Hex(up.Lua); script.Hash() != want {
			t.Errorf("%s: script differs from upstream", name)
		}
		if up.NumberOfKeys != 1 {
			t.Errorf("%s: %d keys", name, up.NumberOfKeys)
		}
	}
}

func testProvider(t *testing.T) *RedisLockingProvider {
	t.Helper()
	url := os.Getenv("MEDUSA_TEST_REDIS_URL")
	if url == "" {
		t.Skip("MEDUSA_TEST_REDIS_URL is not set (e.g. redis://localhost:6379)")
	}
	client := ioredis.New(url, nil)
	prefix := "medusa_lock_test:" + time.Now().Format("150405.000000") + ":"
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisLockingProvider(client, prefix, nil, Options{})
}

func TestRedisAcquireRelease(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	owner := "owner-1"
	other := "owner-2"
	if err := p.Acquire(ctx, []string{"k1"}, lockingtypes.AcquireArgs{OwnerID: &owner, Expire: 10}); err != nil {
		t.Fatal(err)
	}
	if v, _ := p.client.Get(ctx, p.getKeyName("k1")).Result(); v != owner {
		t.Fatalf("value %q", v)
	}
	if ttl, _ := p.client.TTL(ctx, p.getKeyName("k1")).Result(); ttl <= 0 || ttl > 10*time.Second {
		t.Fatalf("ttl %v", ttl)
	}
	err := p.Acquire(ctx, []string{"k1"}, lockingtypes.AcquireArgs{OwnerID: &other})
	var me *common.MedusaError
	if !errors.As(err, &me) || me.Type != common.MedusaErrorTypes.CONFLICT || me.Error() != `Failed to acquire lock for key "k1"` {
		t.Fatalf("got %v", err)
	}
	// no owner: "*", never re-acquired
	if err := p.Acquire(ctx, []string{"k2"}, lockingtypes.AcquireArgs{}); err != nil {
		t.Fatal(err)
	}
	if v, _ := p.client.Get(ctx, p.getKeyName("k2")).Result(); v != "*" {
		t.Fatalf("value %q", v)
	}
	if err := p.Acquire(ctx, []string{"k2"}, lockingtypes.AcquireArgs{}); err == nil {
		t.Fatal("re-acquired an ownerless lock")
	}
	if ok, _ := p.Release(ctx, []string{"k1"}, lockingtypes.ReleaseArgs{OwnerID: &other}); ok {
		t.Fatal("released another owner's lock")
	}
	if ok, _ := p.Release(ctx, []string{"k1", "k2"}, lockingtypes.ReleaseArgs{OwnerID: &owner}); ok {
		t.Fatal("k2 belongs to *")
	}
	if ok, _ := p.Release(ctx, []string{"k2"}, lockingtypes.ReleaseArgs{}); !ok {
		t.Fatal("release of *")
	}
	if err := p.Acquire(ctx, []string{"k3"}, lockingtypes.AcquireArgs{OwnerID: &owner}); err != nil {
		t.Fatal(err)
	}
	if err := p.ReleaseAll(ctx, lockingtypes.ReleaseArgs{OwnerID: &owner}); err != nil {
		t.Fatal(err)
	}
	if n, _ := p.client.Exists(ctx, p.getKeyName("k3")).Result(); n != 0 {
		t.Fatal("k3 not released")
	}
}

func TestRedisExecute(t *testing.T) {
	p := testProvider(t)
	ctx := context.Background()
	var running atomic.Int32
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := p.Execute(ctx, []string{"job"}, func(ctx context.Context) (any, error) {
				if running.Add(1) > 1 {
					t.Error("two jobs hold the same key")
				}
				time.Sleep(100 * time.Millisecond)
				running.Add(-1)
				return nil, nil
			}, lockingtypes.ExecuteArgs{})
			done <- err
		}()
	}
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}

	// timeout while another owner holds the key
	owner := "holder"
	if err := p.Acquire(ctx, []string{"held"}, lockingtypes.AcquireArgs{OwnerID: &owner, Expire: 30}); err != nil {
		t.Fatal(err)
	}
	timeout := 1.0
	_, err := p.Execute(ctx, []string{"held"}, func(ctx context.Context) (any, error) { return nil, nil }, lockingtypes.ExecuteArgs{Timeout: &timeout})
	var me *common.MedusaError
	if !errors.As(err, &me) || me.Type != common.MedusaErrorTypes.CONFLICT || me.Error() != "Timed-out acquiring lock." {
		t.Fatalf("got %v", err)
	}

	// a lost lease aborts the job
	expire := 1.0
	_, err = p.Execute(ctx, []string{"lease"}, func(ctx context.Context) (any, error) {
		_ = p.client.Del(context.Background(), p.getKeyName("lease")).Err()
		select {
		case <-ctx.Done():
		case <-time.After(3 * time.Second):
		}
		return "done", nil
	}, lockingtypes.ExecuteArgs{Expire: &expire})
	if !errors.As(err, &me) || me.Type != common.MedusaErrorTypes.CONFLICT {
		t.Fatalf("got %v", err)
	}
}

func sha1Hex(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
