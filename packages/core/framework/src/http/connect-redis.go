package http

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
)

// This file reproduces connect-redis 5.2 (the express-session store the
// express loader uses when projectConfig.redisUrl is set). Keys are
// <ioredis keyPrefix><prefix><sid>, values are JSON.stringify(session) and
// the expiry is set with SET ... EX / EXPIRE, so Node and Go servers can
// share the same Redis sessions.

// RedisStoreOptions mirrors the connect-redis constructor options.
type RedisStoreOptions struct {
	Client *ioredis.Client
	// Prefix of the keys; nil means the connect-redis default "sess:".
	Prefix *string
	// Ttl in seconds, used when the session cookie has no expiry
	// (connect-redis default: one day).
	Ttl          int
	DisableTTL   bool
	DisableTouch bool
}

// RedisStore mirrors connect-redis RedisStore.
type RedisStore struct {
	client       *ioredis.Client
	prefix       string
	ttl          int
	disableTTL   bool
	disableTouch bool
}

// NewRedisStore mirrors `new RedisStore(options)`.
func NewRedisStore(o RedisStoreOptions) *RedisStore {
	if o.Client == nil {
		panic("A client must be directly provided to the RedisStore")
	}
	s := &RedisStore{client: o.Client, prefix: "sess:", ttl: 86400, disableTTL: o.DisableTTL, disableTouch: o.DisableTouch}
	if o.Prefix != nil {
		s.prefix = *o.Prefix
	}
	if o.Ttl != 0 {
		s.ttl = o.Ttl
	}
	return s
}

// key is the Redis key of a session, including the ioredis keyPrefix.
func (s *RedisStore) key(sid string) string { return s.client.KeyPrefix + s.prefix + sid }

// Get mirrors RedisStore.get.
func (s *RedisStore) Get(sid string) ([]byte, bool, error) {
	data, err := s.client.Get(context.Background(), s.key(sid)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if len(data) == 0 {
		return nil, false, nil
	}
	return data, true, nil
}

// Set mirrors RedisStore.set: SET key value EX ttl, or DEL when the
// session already expired.
func (s *RedisStore) Set(sid string, data []byte, expires time.Time) error {
	ttl := 1
	args := []any{"set", s.key(sid), string(data)}
	if !s.disableTTL {
		ttl = s.getTTL(expires)
		args = append(args, "EX", ttl)
	}
	if ttl > 0 {
		return s.client.Do(context.Background(), args...).Err()
	}
	return s.Destroy(sid)
}

// Touch mirrors RedisStore.touch: EXPIRE key ttl.
func (s *RedisStore) Touch(sid string, _ []byte, expires time.Time) error {
	if s.disableTouch || s.disableTTL {
		return nil
	}
	return s.client.Do(context.Background(), "expire", s.key(sid), s.getTTL(expires)).Err()
}

// Destroy mirrors RedisStore.destroy.
func (s *RedisStore) Destroy(sid string) error {
	return s.client.Del(context.Background(), s.key(sid)).Err()
}

// getTTL mirrors RedisStore._getTTL: seconds until cookie.expires rounded
// up, or the store ttl for sessions without an expiry.
func (s *RedisStore) getTTL(expires time.Time) int {
	if expires.IsZero() {
		return s.ttl
	}
	ms := expires.UnixMilli() - time.Now().UnixMilli()
	return int(math.Ceil(float64(ms) / 1000))
}
