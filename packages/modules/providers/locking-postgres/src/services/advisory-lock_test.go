package services

import (
	"context"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/testdata"
)

// The advisory lock ids must equal upstream's so Node and Go instances
// exclude each other on one database.
func TestHashStringToIntMatchesUpstream(t *testing.T) {
	v, err := testdata.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range v.Keys {
		if got := HashStringToInt(k.Key); got != k.AdvisoryLockKey {
			t.Errorf("%q: got %d, want %d", k.Key, got, k.AdvisoryLockKey)
		}
	}
}

// testDB creates a database with the locking table
// (MEDUSA_TEST_PG_ADMIN_URL).
func testDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := os.Getenv("MEDUSA_TEST_PG_ADMIN_URL")
	if admin == "" {
		t.Skip("MEDUSA_TEST_PG_ADMIN_URL is not set")
	}
	ctx := context.Background()
	name := os.Getenv("MEDUSA_TEST_DB_PREFIX") + "locking_postgres_test"
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
	if _, err := pool.Exec(ctx, `create table if not exists "locking" ("id" text not null, "owner_id" text null, "expiration" timestamptz null, constraint "locking_pkey" primary key ("id"));`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestExecuteHoldsTheAdvisoryLock(t *testing.T) {
	pool := testDB(t)
	p := NewPostgresAdvisoryLockProvider(pool)
	ctx := context.Background()
	key := "cart_01JABCDEF0123456789ABCDEFG"

	// while the job runs, the advisory lock id (as computed upstream) is
	// taken: another session cannot take it
	_, err := p.Execute(ctx, []string{key}, func(ctx context.Context) (any, error) {
		var free bool
		if err := pool.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", int64(2647901747)).Scan(&free); err != nil {
			return nil, err
		}
		if free {
			t.Error("the advisory lock is not held during the job")
		}
		return nil, nil
	}, lockingtypes.ExecuteArgs{})
	if err != nil {
		t.Fatal(err)
	}

	// a second execute waits for the first one
	var running atomic.Int32
	done := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			_, err := p.Execute(ctx, []string{key}, func(ctx context.Context) (any, error) {
				if running.Add(1) > 1 {
					t.Error("two jobs hold the same key")
				}
				time.Sleep(200 * time.Millisecond)
				running.Add(-1)
				return nil, nil
			}, lockingtypes.ExecuteArgs{})
			done <- err
		}()
	}
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}

	// timeout
	release := make(chan struct{})
	started := make(chan struct{})
	go func() {
		_, _ = p.Execute(ctx, []string{key}, func(ctx context.Context) (any, error) {
			close(started)
			<-release
			return nil, nil
		}, lockingtypes.ExecuteArgs{})
	}()
	<-started
	timeout := 1.0
	_, err = p.Execute(ctx, []string{key}, func(ctx context.Context) (any, error) { return nil, nil }, lockingtypes.ExecuteArgs{Timeout: &timeout})
	close(release)
	if err == nil || err.Error() != "Timed-out acquiring lock." {
		t.Fatalf("got %v", err)
	}
}

func TestAcquireReleaseRows(t *testing.T) {
	pool := testDB(t)
	p := NewPostgresAdvisoryLockProvider(pool)
	ctx := context.Background()
	owner := "owner-1"
	other := "owner-2"

	if err := p.Acquire(ctx, []string{"k1"}, lockingtypes.AcquireArgs{OwnerID: &owner, Expire: 10}); err != nil {
		t.Fatal(err)
	}
	// same owner refreshes
	if err := p.Acquire(ctx, []string{"k1"}, lockingtypes.AcquireArgs{OwnerID: &owner, Expire: 10}); err != nil {
		t.Fatal(err)
	}
	if err := p.Acquire(ctx, []string{"k1"}, lockingtypes.AcquireArgs{OwnerID: &other}); err == nil || err.Error() != `Failed to acquire lock for key "k1"` {
		t.Fatalf("got %v", err)
	}
	// a lock without owner can never be re-acquired
	if err := p.Acquire(ctx, []string{"k2"}, lockingtypes.AcquireArgs{}); err != nil {
		t.Fatal(err)
	}
	if err := p.Acquire(ctx, []string{"k2"}, lockingtypes.AcquireArgs{}); err == nil || !strings.Contains(err.Error(), `"k2"`) {
		t.Fatalf("got %v", err)
	}
	var exp *time.Time
	if err := pool.QueryRow(ctx, "SELECT expiration FROM locking WHERE id = 'k1'").Scan(&exp); err != nil || exp == nil {
		t.Fatalf("expiration %v %v", exp, err)
	}
	if ok, _ := p.Release(ctx, []string{"k1"}, lockingtypes.ReleaseArgs{OwnerID: &other}); ok {
		t.Fatal("released another owner's lock")
	}
	if ok, err := p.Release(ctx, []string{"k1"}, lockingtypes.ReleaseArgs{OwnerID: &owner}); !ok || err != nil {
		t.Fatalf("release: %v %v", ok, err)
	}
	if ok, _ := p.Release(ctx, []string{"k2"}, lockingtypes.ReleaseArgs{}); !ok {
		t.Fatal("release of an ownerless lock")
	}
	if err := p.Acquire(ctx, []string{"k3", "k4"}, lockingtypes.AcquireArgs{OwnerID: &owner}); err != nil {
		t.Fatal(err)
	}
	if err := p.ReleaseAll(ctx, lockingtypes.ReleaseArgs{OwnerID: &owner}); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = pool.QueryRow(ctx, "SELECT count(*) FROM locking").Scan(&n)
	if n != 0 {
		t.Fatalf("%d rows left", n)
	}
}
