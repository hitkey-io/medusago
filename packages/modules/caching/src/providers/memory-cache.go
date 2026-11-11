// Package providers mirrors @medusajs/caching/src/providers: the built in
// memory caching provider.
package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/hitkey-io/medusago/packages/modules/caching/src/types"
)

// MemoryIdentifier mirrors MemoryCachingProvider.identifier.
const MemoryIdentifier = "cache-memory"

const threeHundredMB = 300 * 1024 * 1024

func formatBytes(bytes int) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%d KB", int(math.Round(float64(bytes)/1024)))
	}
	return fmt.Sprintf("%d MB", int(math.Round(float64(bytes)/1024/1024)))
}

type entry struct {
	value   any
	expires time.Time // zero: no expiry
}

// MemoryCachingProvider mirrors MemoryCachingProvider (node-cache with tag
// indexes): entries keyed by the hashed key, tag -> keys and key -> tags
// indexes of hashed tags, the entries' options and an approximate size.
type MemoryCachingProvider struct {
	logger *slog.Logger
	hasher func(string) string

	ttl         int
	maxKeys     int
	checkPeriod int
	maxSize     int

	mu                     sync.Mutex
	entries                map[string]*entry
	tagIndex               map[string]map[string]bool
	tagIndexOrder          map[string][]string
	keyTags                map[string]map[string]bool
	entryOptions           map[string]*bool
	keySizes               map[string]int
	approximateMemoryUsage int
	stop                   chan struct{}
	stopOnce               sync.Once
}

// NewMemoryCachingProvider mirrors the constructor (ttl 3600, maxKeys
// 25000, checkPeriod 60, maxSize 300 MB).
func NewMemoryCachingProvider(logger *slog.Logger, hasher func(string) string, opts types.MemoryCacheModuleOptions) *MemoryCachingProvider {
	if logger == nil {
		logger = slog.Default()
	}
	p := &MemoryCachingProvider{
		logger: logger, hasher: hasher, ttl: 3600, maxKeys: 25000, checkPeriod: 60, maxSize: threeHundredMB,
		entries: map[string]*entry{}, tagIndex: map[string]map[string]bool{}, tagIndexOrder: map[string][]string{},
		keyTags: map[string]map[string]bool{}, entryOptions: map[string]*bool{}, keySizes: map[string]int{},
		stop: make(chan struct{}),
	}
	if opts.TTL != nil {
		p.ttl = *opts.TTL
	}
	if opts.MaxKeys != nil {
		p.maxKeys = *opts.MaxKeys
	}
	if opts.CheckPeriod != nil {
		p.checkPeriod = *opts.CheckPeriod
	}
	if opts.MaxSize != nil {
		p.maxSize = *opts.MaxSize
	}
	if p.checkPeriod > 0 {
		go p.checkLoop()
	}
	return p
}

// checkLoop mirrors node-cache's checkperiod: expired entries are removed.
func (p *MemoryCachingProvider) checkLoop() {
	t := time.NewTicker(time.Duration(p.checkPeriod) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-p.stop:
			return
		case <-t.C:
			p.mu.Lock()
			now := time.Now()
			for k, e := range p.entries {
				if !e.expires.IsZero() && !now.Before(e.expires) {
					p.del(k)
				}
			}
			p.mu.Unlock()
		}
	}
}

// Shutdown mirrors shutdown (cacheClient.close()).
func (p *MemoryCachingProvider) Shutdown(context.Context) error {
	p.stopOnce.Do(func() { close(p.stop) })
	return nil
}

// del deletes an entry and its tag references (node-cache "del" event).
func (p *MemoryCachingProvider) del(key string) {
	if _, ok := p.entries[key]; !ok {
		return
	}
	delete(p.entries, key)
	p.cleanupTagReferences(key)
}

func (p *MemoryCachingProvider) cleanupTagReferences(key string) {
	if tags, ok := p.keyTags[key]; ok {
		for tag := range tags {
			if keys := p.tagIndex[tag]; keys != nil {
				delete(keys, key)
				order := p.tagIndexOrder[tag]
				for i, k := range order {
					if k == key {
						p.tagIndexOrder[tag] = append(order[:i:i], order[i+1:]...)
						break
					}
				}
				if len(keys) == 0 {
					delete(p.tagIndex, tag)
					delete(p.tagIndexOrder, tag)
				}
			}
		}
		delete(p.keyTags, key)
	}
	if size, ok := p.keySizes[key]; ok && size != 0 {
		p.approximateMemoryUsage -= size
		delete(p.keySizes, key)
	}
	delete(p.entryOptions, key)
}

// lookup returns a live entry (an expired one is removed, like node-cache's
// _check).
func (p *MemoryCachingProvider) lookup(key string) (any, bool) {
	e, ok := p.entries[key]
	if !ok {
		return nil, false
	}
	if !e.expires.IsZero() && !time.Now().Before(e.expires) {
		p.del(key)
		return nil, false
	}
	return e.value, true
}

// Get mirrors get({ key, tags }).
func (p *MemoryCachingProvider) Get(_ context.Context, key string, tags []string) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if key != "" {
		v, _ := p.lookup(p.hasher(key))
		return v, nil
	}
	if len(tags) > 0 {
		var keys []string
		seen := map[string]bool{}
		for _, tag := range tags {
			for _, k := range p.tagIndexOrder[p.hasher(tag)] {
				if !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
		}
		results := []any{}
		for _, k := range keys {
			if v, ok := p.lookup(k); ok {
				results = append(results, v)
			}
		}
		return results, nil
	}
	return nil, nil
}

func jsonLen(v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		return 0
	}
	return len(b)
}

// Set mirrors set({ key, data, ttl, tags, options }).
func (p *MemoryCachingProvider) Set(_ context.Context, o types.ProviderSetOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.approximateMemoryUsage > p.maxSize {
		p.logger.Warn(fmt.Sprintf("Cache is full. Current usage: %s, Max: %s. Skipping cache entry.", formatBytes(p.approximateMemoryUsage), formatBytes(p.maxSize)))
		return nil
	}
	hashedKey := p.hasher(o.Key)
	var hashedTags []string
	for _, t := range o.Tags {
		hashedTags = append(hashedTags, p.hasher(t))
	}
	totalSize := jsonLen(o.Data) + len(hashedKey)
	if len(hashedTags) > 0 {
		totalSize += jsonLen(hashedTags)
	}
	ttl := p.ttl
	if o.TTL != nil {
		ttl = *o.TTL
	}
	// node-cache: a new key beyond maxKeys throws ECACHEFULL
	if _, exists := p.entries[hashedKey]; !exists && p.maxKeys > -1 && len(p.entries) >= p.maxKeys {
		return errors.New("Cache max keys amount exceeded")
	}
	e := &entry{value: o.Data}
	if ttl > 0 {
		e.expires = time.Now().Add(time.Duration(ttl) * time.Second)
	}
	p.entries[hashedKey] = e
	if len(hashedTags) > 0 {
		p.cleanupTagReferences(hashedKey)
		set := map[string]bool{}
		for _, t := range hashedTags {
			set[t] = true
			if p.tagIndex[t] == nil {
				p.tagIndex[t] = map[string]bool{}
			}
			if !p.tagIndex[t][hashedKey] {
				p.tagIndex[t][hashedKey] = true
				p.tagIndexOrder[t] = append(p.tagIndexOrder[t], hashedKey)
			}
		}
		p.keyTags[hashedKey] = set
	}
	if o.HasOptions && o.AutoInvalidate != nil {
		v := *o.AutoInvalidate
		p.entryOptions[hashedKey] = &v
	}
	existing := p.keySizes[hashedKey]
	p.approximateMemoryUsage = p.approximateMemoryUsage - existing + totalSize
	p.keySizes[hashedKey] = totalSize
	return nil
}

// Clear mirrors clear({ key, tags, options }).
func (p *MemoryCachingProvider) Clear(_ context.Context, o types.ProviderClearOptions) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if o.Key != "" {
		p.del(p.hasher(o.Key))
		return nil
	}
	if len(o.Tags) == 0 {
		return nil
	}
	for _, t := range o.Tags {
		if t == "*" {
			p.entries = map[string]*entry{}
			p.tagIndex = map[string]map[string]bool{}
			p.tagIndexOrder = map[string][]string{}
			p.keyTags = map[string]map[string]bool{}
			p.entryOptions = map[string]*bool{}
			p.keySizes = map[string]int{}
			p.approximateMemoryUsage = 0
			return nil
		}
	}
	var keys []string
	seen := map[string]bool{}
	for _, t := range o.Tags {
		for _, k := range p.tagIndexOrder[p.hasher(t)] {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	if !o.Options {
		for _, k := range keys {
			p.del(k)
			p.cleanupTagReferences(k)
		}
		return nil
	}
	if o.AutoInvalidate != nil && *o.AutoInvalidate {
		for _, k := range keys {
			auto := true
			if v := p.entryOptions[k]; v != nil {
				auto = *v
			}
			if auto {
				p.del(k)
				p.cleanupTagReferences(k)
			}
		}
	}
	return nil
}
