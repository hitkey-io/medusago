// Package loaders mirrors @medusajs/locking-redis/src/loaders.
package loaders

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	modulessdk "github.com/hitkey-io/medusago/packages/core/types/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/modules/providers/locking-redis/src/services"
)

// optNumber mirrors `+options.x` for the numeric options (nil when not a
// number).
func optNumber(opts map[string]any, key string) *float64 {
	v, ok := opts[key]
	if !ok || v == nil {
		return nil
	}
	switch x := v.(type) {
	case float64:
		return &x
	case int:
		f := float64(x)
		return &f
	case int64:
		f := float64(x)
		return &f
	case string:
		if f, err := strconv.ParseFloat(x, 64); err == nil {
			return &f
		}
	}
	return nil
}

// Load mirrors the provider loader: a Redis connection from redisUrl and
// redisOptions, and the key prefix (namespace, default "medusa_lock:").
func Load(opts modulessdk.ProviderOptions) (*services.RedisLockingProvider, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	redisURL, _ := opts.Options["redisUrl"].(string)
	if redisURL == "" {
		return nil, errors.New(`No "redisUrl" provided in "locking" module, "locking-redis" provider options. It is required for the "locking-redis" Module provider.`)
	}
	redisOptions, _ := opts.Options["redisOptions"].(map[string]any)
	client := ioredis.New(redisURL, redisOptions)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error(fmt.Sprintf(`An error occurred while connecting to Redis in provider "locking-redis": %v`, err))
	} else {
		logger.Info(`Connection to Redis in "locking-redis" provider established`)
	}
	prefix := "medusa_lock:"
	if ns, ok := opts.Options["namespace"].(string); ok {
		prefix = ns
	}
	return services.NewRedisLockingProvider(client, prefix, logger, services.Options{
		RedisURL:             redisURL,
		RedisOptions:         redisOptions,
		WaitLockingTimeout:   optNumber(opts.Options, "waitLockingTimeout"),
		DefaultRetryInterval: optNumber(opts.Options, "defaultRetryInterval"),
		MaximumRetryInterval: optNumber(opts.Options, "maximumRetryInterval"),
		BackoffFactor:        optNumber(opts.Options, "backoffFactor"),
	}), nil
}
