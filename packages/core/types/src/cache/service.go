// Package cache mirrors @medusajs/types/src/cache: the cache module
// interface (Modules.CACHE).
package cache

import "context"

// ICacheService mirrors ICacheService. Get returns nil when the key is
// missing or expired. A nil ttl uses the module TTL; 0 does not cache.
// Values go through JSON with the Redis implementation (objects come back
// as *dal.Record, dates as ISO strings).
type ICacheService interface {
	Get(ctx context.Context, key string) (any, error)
	Set(ctx context.Context, key string, data any, ttl *float64) error
	Invalidate(ctx context.Context, key string) error
}

// TTL returns a pointer to a number of seconds (helper for Set).
func TTL(seconds float64) *float64 { return &seconds }
