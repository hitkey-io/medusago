package providers

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
)

func TestInMemoryAcquireRelease(t *testing.T) {
	p := NewInMemoryLockingProvider()
	ctx := context.Background()
	owner, other := "owner-1", "owner-2"

	if err := p.Acquire(ctx, []string{"k"}, lockingtypes.AcquireArgs{OwnerID: &owner}); err != nil {
		t.Fatal(err)
	}
	if err := p.Acquire(ctx, []string{"k"}, lockingtypes.AcquireArgs{OwnerID: &owner, Expire: 10}); err != nil {
		t.Fatal(err)
	}
	if err := p.Acquire(ctx, []string{"k"}, lockingtypes.AcquireArgs{OwnerID: &other}); err == nil || err.Error() != `Failed to acquire lock for key "k"` {
		t.Fatalf("got %v", err)
	}
	// a lock without owner cannot be acquired again
	if err := p.Acquire(ctx, []string{"n"}, lockingtypes.AcquireArgs{}); err != nil {
		t.Fatal(err)
	}
	if err := p.Acquire(ctx, []string{"n"}, lockingtypes.AcquireArgs{}); err == nil {
		t.Fatal("re-acquired an ownerless lock")
	}
	if ok, _ := p.Release(ctx, []string{"k"}, lockingtypes.ReleaseArgs{OwnerID: &other}); ok {
		t.Fatal("released another owner's lock")
	}
	// undefined owner releases any lock
	if ok, _ := p.Release(ctx, []string{"k", "n"}, lockingtypes.ReleaseArgs{}); !ok {
		t.Fatal("release")
	}
	if ok, _ := p.Release(ctx, []string{"k"}, lockingtypes.ReleaseArgs{}); ok {
		t.Fatal("released a missing lock")
	}

	// expired locks are taken over
	if err := p.Acquire(ctx, []string{"e"}, lockingtypes.AcquireArgs{OwnerID: &owner, Expire: 0.05}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if err := p.Acquire(ctx, []string{"e"}, lockingtypes.AcquireArgs{OwnerID: &other}); err != nil {
		t.Fatal(err)
	}
	if err := p.ReleaseAll(ctx, lockingtypes.ReleaseArgs{OwnerID: &other}); err != nil {
		t.Fatal(err)
	}
	if err := p.Acquire(ctx, []string{"e"}, lockingtypes.AcquireArgs{OwnerID: &owner}); err != nil {
		t.Fatal(err)
	}
}

func TestInMemoryExecute(t *testing.T) {
	p := NewInMemoryLockingProvider()
	ctx := context.Background()
	var running atomic.Int32
	done := make(chan error, 3)
	for i := 0; i < 3; i++ {
		go func() {
			_, err := p.Execute(ctx, []string{"a", "b"}, func(ctx context.Context) (any, error) {
				if running.Add(1) > 1 {
					t.Error("two jobs hold the same keys")
				}
				time.Sleep(50 * time.Millisecond)
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
	if len(p.locks) != 0 {
		t.Fatalf("%d locks left", len(p.locks))
	}

	// the step lock (acquire) blocks execute until its timeout
	if err := p.Acquire(ctx, []string{"cart"}, lockingtypes.AcquireArgs{}); err != nil {
		t.Fatal(err)
	}
	timeout := 1.0
	start := time.Now()
	_, err := p.Execute(ctx, []string{"cart"}, func(ctx context.Context) (any, error) { return nil, nil }, lockingtypes.ExecuteArgs{Timeout: &timeout})
	if err == nil || err.Error() != "Timed-out acquiring lock." || time.Since(start) < time.Second {
		t.Fatalf("got %v after %v", err, time.Since(start))
	}
	// released: a queued execute proceeds
	go func() {
		time.Sleep(100 * time.Millisecond)
		_, _ = p.Release(ctx, []string{"cart"}, lockingtypes.ReleaseArgs{})
	}()
	out, err := p.Execute(ctx, []string{"cart"}, func(ctx context.Context) (any, error) { return "ok", nil }, lockingtypes.ExecuteArgs{})
	if err != nil || out != "ok" {
		t.Fatalf("got %v %v", out, err)
	}
}
