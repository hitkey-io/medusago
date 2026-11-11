// Package services mirrors @medusajs/cache-inmemory/src/services.
package services

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"time"

	cachetypes "github.com/hitkey-io/medusago/packages/core/types/src/cache"
)

// DefaultTTL mirrors DEFAULT_TTL (seconds).
const DefaultTTL = 30

type cacheRecord struct {
	data any
	// expire in unix milliseconds
	expire int64
	timer  *time.Timer
}

// InMemoryCacheService mirrors InMemoryCacheService: a process-local store
// whose records are dropped when their ttl elapses.
type InMemoryCacheService struct {
	ttl   float64
	mu    sync.Mutex
	store map[string]*cacheRecord
}

var _ cachetypes.ICacheService = (*InMemoryCacheService)(nil)

// NewInMemoryCacheService mirrors the constructor (options.ttl ?? 30).
func NewInMemoryCacheService(ttl *float64) *InMemoryCacheService {
	t := float64(DefaultTTL)
	if ttl != nil {
		t = *ttl
	}
	return &InMemoryCacheService{ttl: t, store: map[string]*cacheRecord{}}
}

// Get mirrors get: nil when missing or expired.
func (s *InMemoryCacheService) Get(ctx context.Context, key string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.store[key]
	if !ok || r.expire < time.Now().UnixMilli() {
		return nil, nil
	}
	return r.data, nil
}

// Set mirrors set: ttl 0 does not cache; the record is invalidated when the
// ttl elapses.
func (s *InMemoryCacheService) Set(ctx context.Context, key string, data any, ttl *float64) error {
	t := s.ttl
	if ttl != nil {
		t = *ttl
	}
	if t == 0 {
		return nil
	}
	d := time.Duration(t * float64(time.Second))
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.store[key]; ok && old.timer != nil {
		old.timer.Stop()
	}
	r := &cacheRecord{data: data, expire: time.Now().UnixMilli() + int64(t*1000)}
	r.timer = time.AfterFunc(d, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if cur, ok := s.store[key]; ok && cur == r {
			delete(s.store, key)
		}
	})
	s.store[key] = r
	return nil
}

// Invalidate mirrors invalidate: a key with "*" is a pattern (the first "*"
// becomes ".*", unanchored).
func (s *InMemoryCacheService) Invalidate(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := []string{key}
	if strings.Contains(key, "*") {
		re, err := regexp.Compile(strings.Replace(key, "*", ".*", 1))
		if err != nil {
			return err
		}
		keys = keys[:0]
		for k := range s.store {
			if re.MatchString(k) {
				keys = append(keys, k)
			}
		}
	}
	for _, k := range keys {
		if r, ok := s.store[k]; ok {
			if r.timer != nil {
				r.timer.Stop()
			}
			delete(s.store, k)
		}
	}
	return nil
}

// Clear mirrors clear.
func (s *InMemoryCacheService) Clear(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, r := range s.store {
		if r.timer != nil {
			r.timer.Stop()
		}
		delete(s.store, k)
	}
	return nil
}
