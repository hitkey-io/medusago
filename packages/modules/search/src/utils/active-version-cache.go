package utils

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
)

// ActiveIndexVersion mirrors ActiveIndexVersion.
type ActiveIndexVersion struct {
	PhysicalName string
	Provider     string
	Version      int
}

const (
	softTTL = 30 * time.Second
	hardTTL = 2 * time.Minute
)

// ActiveIndexVersionCache mirrors ActiveIndexVersionCache: which physical
// index serves reads for every logical index, fetched in one bulk read,
// fresh for 30s, served stale while a background refresh runs, never past
// 2 minutes since the last successful fetch.
type ActiveIndexVersionCache struct {
	fetchAll func(ctx context.Context) (map[string]ActiveIndexVersion, error)

	mu         sync.Mutex
	values     map[string]ActiveIndexVersion
	fetchedAt  time.Time
	refreshing chan struct{}
	refreshErr error
}

// NewActiveIndexVersionCache creates the cache.
func NewActiveIndexVersionCache(fetchAll func(ctx context.Context) (map[string]ActiveIndexVersion, error)) *ActiveIndexVersionCache {
	return &ActiveIndexVersionCache{fetchAll: fetchAll, values: map[string]ActiveIndexVersion{}}
}

// Get mirrors get(name, { fresh }).
func (c *ActiveIndexVersionCache) Get(ctx context.Context, name string, fresh bool) (ActiveIndexVersion, error) {
	c.mu.Lock()
	fetchedAt := c.fetchedAt
	c.mu.Unlock()
	age := time.Since(fetchedAt)
	if fresh || fetchedAt.IsZero() || age >= hardTTL {
		if err := c.refresh(ctx); err != nil {
			return ActiveIndexVersion{}, err
		}
	} else if age >= softTTL {
		go func() { _ = c.refresh(context.Background()) }()
	}
	c.mu.Lock()
	value, ok := c.values[name]
	c.mu.Unlock()
	if !ok {
		return ActiveIndexVersion{}, common.NewMedusaError(common.MedusaErrorTypes.NOT_FOUND, fmt.Sprintf(`Search index "%s" has no active version yet. It has to be migrated and seeded first.`, name))
	}
	return value, nil
}

// refresh shares one fetch between concurrent callers.
func (c *ActiveIndexVersionCache) refresh(ctx context.Context) error {
	c.mu.Lock()
	if wait := c.refreshing; wait != nil {
		c.mu.Unlock()
		<-wait
		c.mu.Lock()
		err := c.refreshErr
		c.mu.Unlock()
		return err
	}
	done := make(chan struct{})
	c.refreshing = done
	c.mu.Unlock()

	values, err := c.fetchAll(ctx)

	c.mu.Lock()
	if err == nil {
		c.values = values
		c.fetchedAt = time.Now()
	}
	c.refreshErr = err
	c.refreshing = nil
	c.mu.Unlock()
	close(done)
	return err
}

// settle waits for a fetch in flight.
func (c *ActiveIndexVersionCache) settle() {
	c.mu.Lock()
	wait := c.refreshing
	c.mu.Unlock()
	if wait != nil {
		<-wait
	}
}

// Set mirrors set(name, value).
func (c *ActiveIndexVersionCache) Set(name string, value ActiveIndexVersion) {
	c.settle()
	c.mu.Lock()
	c.values[name] = value
	c.mu.Unlock()
}

// Invalidate mirrors invalidate().
func (c *ActiveIndexVersionCache) Invalidate() {
	c.settle()
	c.mu.Lock()
	c.fetchedAt = time.Time{}
	c.values = map[string]ActiveIndexVersion{}
	c.mu.Unlock()
}
