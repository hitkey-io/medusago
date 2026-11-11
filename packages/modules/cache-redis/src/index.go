// Package cacheredis mirrors @medusajs/cache-redis: the Redis cache module
// (Modules.CACHE).
package cacheredis

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/modules/cache-redis/src/services"
)

func numberOption(opts map[string]any, key string) *float64 {
	switch v := opts[key].(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case int64:
		f := float64(v)
		return &f
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return &f
		}
	}
	return nil
}

// Module is the module definition (options: redisUrl, redisOptions, ttl,
// namespace).
var Module = modulessdk.ModuleExports{
	Service: func(opts modulessdk.ServiceOptions) (any, error) {
		logger := opts.Logger
		if logger == nil {
			logger = slog.Default()
		}
		// loaders/index.ts
		redisURL, _ := opts.Options["redisUrl"].(string)
		if redisURL == "" {
			return nil, errors.New("No `redisUrl` provided in `cacheService` module options. It is required for the Redis Cache Module.")
		}
		redisOptions, _ := opts.Options["redisOptions"].(map[string]any)
		client := ioredis.New(redisURL, redisOptions)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			logger.Error(fmt.Sprintf("An error occurred while connecting to Redis in module 'cache-redis': %v", err))
		} else {
			logger.Info("Connection to Redis in module 'cache-redis' established")
		}
		namespace, _ := opts.Options["namespace"].(string)
		return services.NewRedisCacheService(client, numberOption(opts.Options, "ttl"), namespace), nil
	},
}
