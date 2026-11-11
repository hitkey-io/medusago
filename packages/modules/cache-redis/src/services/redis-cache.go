// Package services mirrors @medusajs/cache-redis/src/services.
package services

import (
	"context"
	"errors"
	"strconv"

	"github.com/redis/go-redis/v9"

	cachetypes "github.com/hitkey-io/medusago/packages/core/types/src/cache"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
)

// DefaultNamespace mirrors DEFAULT_NAMESPACE.
const DefaultNamespace = "medusa"

// DefaultCacheTime mirrors DEFAULT_CACHE_TIME (seconds).
const DefaultCacheTime = 30

// RedisCacheService mirrors RedisCacheService: JSON values under
// `<namespace>:<key>` with an EX expiry, so Node and Go instances share them.
type RedisCacheService struct {
	redis     *ioredis.Client
	ttl       float64
	namespace string
}

var _ cachetypes.ICacheService = (*RedisCacheService)(nil)

// NewRedisCacheService mirrors the constructor (ttl ?? 30, namespace ||
// "medusa").
func NewRedisCacheService(client *ioredis.Client, ttl *float64, namespace string) *RedisCacheService {
	t := float64(DefaultCacheTime)
	if ttl != nil {
		t = *ttl
	}
	if namespace == "" {
		namespace = DefaultNamespace
	}
	return &RedisCacheService{redis: client, ttl: t, namespace: namespace}
}

// OnApplicationShutdown mirrors __hooks.onApplicationShutdown.
func (s *RedisCacheService) OnApplicationShutdown(ctx context.Context) error {
	return s.redis.Close()
}

// getCacheKey mirrors getCacheKey (the ioredis keyPrefix applies on top).
func (s *RedisCacheService) getCacheKey(key string) string {
	if s.namespace != "" {
		return s.namespace + ":" + key
	}
	return key
}

// Set mirrors set: JSON.stringify(data) with EX ttl (0 does not cache).
func (s *RedisCacheService) Set(ctx context.Context, key string, data any, ttl *float64) error {
	t := s.ttl
	if ttl != nil {
		t = *ttl
	}
	if t == 0 {
		return nil
	}
	raw, err := dal.MarshalValue(data)
	if err != nil {
		return err
	}
	return s.redis.Do(ctx, "set", s.redis.KeyPrefix+s.getCacheKey(key), string(raw), "EX", jsNumber(t)).Err()
}

// Get mirrors get: the parsed JSON value, nil when missing; a value that
// does not parse is unlinked.
func (s *RedisCacheService) Get(ctx context.Context, key string) (any, error) {
	cacheKey := s.redis.KeyPrefix + s.getCacheKey(key)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		_ = s.redis.Unlink(ctx, cacheKey).Err()
		return nil, nil
	}
	if cached == "" {
		return nil, nil
	}
	v, err := dal.DecodeOrderedJSON([]byte(cached))
	if err != nil {
		_ = s.redis.Unlink(ctx, cacheKey).Err()
		return nil, nil
	}
	return v, nil
}

// Invalidate mirrors invalidate: SCAN MATCH `<namespace>:<key>` (a
// pattern) and unlink the matches.
func (s *RedisCacheService) Invalidate(ctx context.Context, key string) error {
	pattern := s.getCacheKey(key)
	var cursor uint64
	for {
		// ioredis does not prefix SCAN patterns; UNLINK is prefixed
		keys, next, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		cursor = next
		if len(keys) > 0 {
			pipe := s.redis.Pipeline()
			for _, k := range keys {
				pipe.Unlink(ctx, s.redis.KeyPrefix+k)
			}
			if _, err := pipe.Exec(ctx); err != nil {
				return err
			}
		}
		if cursor == 0 {
			return nil
		}
	}
}

func jsNumber(n float64) string { return strconv.FormatFloat(n, 'f', -1, 64) }
