// Package loaders mirrors @medusajs/caching-redis/src/loaders.
package loaders

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	cachingloaders "github.com/hitkey-io/medusago/packages/modules/caching/src/loaders"
	"github.com/hitkey-io/medusago/packages/modules/providers/caching-redis/src/services"
)

// moduleOptionKeys are the module options that are not ioredis options.
var moduleOptionKeys = map[string]bool{"redisUrl": true, "redisOptions": true, "ttl": true, "prefix": true, "compressionThreshold": true}

func intOpt(m map[string]any, k string) *int {
	switch v := m[k].(type) {
	case float64:
		n := int(v)
		return &n
	case int:
		return &v
	case int64:
		n := int(v)
		return &n
	}
	return nil
}

// Load mirrors the connection and hash loaders: a Redis client from
// redisUrl and redisOptions (connectTimeout 10 s, commandTimeout 5 s,
// maxRetriesPerRequest 3, the deprecated top level ioredis options), the
// key prefix ("mc:") and the xxhash hasher.
func Load(opts modulessdk.ProviderOptions) (*services.RedisCachingProvider, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	redisURL, _ := opts.Options["redisUrl"].(string)
	if redisURL == "" {
		return nil, errors.New("[caching-redis] redisUrl is required")
	}
	newRedisOptions, _ := opts.Options["redisOptions"].(map[string]any)
	deprecated := map[string]any{}
	for k, v := range opts.Options {
		if !moduleOptionKeys[k] {
			deprecated[k] = v
		}
	}
	if newRedisOptions == nil && len(deprecated) > 0 {
		logger.Warn("[caching-redis] Passing ioredis options at the top level of the module options is deprecated. Please use `redisOptions` instead for consistency with other modules.")
	}
	redisOptions := map[string]any{
		"connectTimeout": 10000, "commandTimeout": 5000, "lazyConnect": true, "maxRetriesPerRequest": 3,
		"enableOfflineQueue": true, "connectionName": "medusa-cache-redis",
	}
	extra := newRedisOptions
	if extra == nil {
		extra = deprecated
	}
	for k, v := range extra {
		redisOptions[k] = v
	}
	client := ioredis.New(redisURL, redisOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn(fmt.Sprintf("Redis cache connection test failed: %s, but continuing with lazy connection", err.Error()))
	} else {
		logger.Info("Redis cache connection test successful")
	}
	prefix := "mc:"
	if p, ok := opts.Options["prefix"].(string); ok {
		prefix = p
	}
	return services.NewRedisCachingProvider(client.Client, prefix, cachingloaders.Hasher, logger, services.Options{
		TTL: intOpt(opts.Options, "ttl"), CompressionThreshold: intOpt(opts.Options, "compressionThreshold"),
	}), nil
}
