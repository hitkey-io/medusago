// Package services mirrors @medusajs/locking-redis/src/services.
package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	lockingtypes "github.com/hitkey-io/medusago/packages/core/types/src/locking"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/ioredis"
	"github.com/hitkey-io/medusago/packages/modules/locking/src/providers"
)

// Identifier mirrors RedisLockingProvider.identifier.
const Identifier = "locking-redis"

// DefaultLockExpiration mirrors DEFAULT_LOCK_EXPIRATION (seconds): the lease
// of the keys taken by execute, renewed while the job runs.
const DefaultLockExpiration = 60

// The Lua commands defined on the ioredis client (same scripts, so Node and
// Go instances interoperate on the same keys and values).
var (
	acquireLockScript = redis.NewScript(`
        local key = KEYS[1]
        local ownerId = ARGV[1]
        local ttl = tonumber(ARGV[2])
        local awaitQueue = ARGV[3] == 'true'

        local setArgs = {key, ownerId, 'NX'}
        if ttl > 0 then
            table.insert(setArgs, 'EX')
            table.insert(setArgs, ttl)
        end

        local setResult = redis.call('SET', unpack(setArgs))

        if setResult then
            return 1
        elseif not awaitQueue then
            -- Key already exists; retrieve the current ownerId
            local currentOwnerId = redis.call('GET', key)
            if currentOwnerId == '*' then
              return 0
            elseif currentOwnerId == ownerId then
                setArgs = {key, ownerId, 'XX'}
                if ttl > 0 then
                    table.insert(setArgs, 'EX')
                    table.insert(setArgs, ttl)
                end
                redis.call('SET', unpack(setArgs))
                return 1
            else
                return 0
            end
        else
            return 0
        end

      `)
	extendLockScript = redis.NewScript(`
        local key = KEYS[1]
        local ownerId = ARGV[1]
        local ttl = tonumber(ARGV[2])

        if redis.call('GET', key) == ownerId then
          return redis.call('EXPIRE', key, ttl)
        else
          return 0
        end
      `)
	releaseLockScript = redis.NewScript(`
        local key = KEYS[1]
        local ownerId = ARGV[1]

        if redis.call('GET', key) == ownerId then
          return redis.call('DEL', key)
        else
          return 0
        end
      `)
)

// Options mirrors the provider options (RedisCacheModuleOptions of
// locking-redis).
type Options struct {
	RedisURL             string
	RedisOptions         map[string]any
	Namespace            *string
	WaitLockingTimeout   *float64
	DefaultRetryInterval *float64
	MaximumRetryInterval *float64
	BackoffFactor        *float64
}

// RedisLockingProvider mirrors RedisLockingProvider.
type RedisLockingProvider struct {
	client               *ioredis.Client
	keyNamePrefix        string
	logger               *slog.Logger
	waitLockingTimeout   float64
	defaultRetryInterval float64
	maximumRetryInterval float64
	backoffFactor        float64
}

var _ lockingtypes.ILockingProvider = (*RedisLockingProvider)(nil)

// NewRedisLockingProvider mirrors the constructor (the client and the
// prefix come from the provider loader).
func NewRedisLockingProvider(client *ioredis.Client, prefix string, logger *slog.Logger, opts Options) *RedisLockingProvider {
	if logger == nil {
		logger = slog.Default()
	}
	p := &RedisLockingProvider{client: client, keyNamePrefix: prefix, logger: logger,
		waitLockingTimeout: 5, defaultRetryInterval: 20, maximumRetryInterval: 1000, backoffFactor: 2}
	if opts.WaitLockingTimeout != nil && !math.IsNaN(*opts.WaitLockingTimeout) {
		p.waitLockingTimeout = *opts.WaitLockingTimeout
	}
	if opts.DefaultRetryInterval != nil && !math.IsNaN(*opts.DefaultRetryInterval) {
		p.defaultRetryInterval = *opts.DefaultRetryInterval
	}
	if opts.MaximumRetryInterval != nil && !math.IsNaN(*opts.MaximumRetryInterval) {
		p.maximumRetryInterval = *opts.MaximumRetryInterval
	}
	if opts.BackoffFactor != nil && !math.IsNaN(*opts.BackoffFactor) {
		p.backoffFactor = *opts.BackoffFactor
	}
	return p
}

// getKeyName mirrors getKeyName (the ioredis keyPrefix applies on top).
func (p *RedisLockingProvider) getKeyName(key string) string {
	return p.client.KeyPrefix + p.keyNamePrefix + key
}

func conflict(msg string) error {
	return common.NewMedusaError(common.MedusaErrorTypes.CONFLICT, msg)
}

// Execute mirrors execute: the keys are awaited (exponential backoff) for at
// most timeout seconds and held with a lease renewed every third of expire;
// the job's context is cancelled when the lease is lost.
func (p *RedisLockingProvider) Execute(ctx context.Context, keys []string, job lockingtypes.Job, args lockingtypes.ExecuteArgs) (any, error) {
	timeoutSeconds := providers.ExecuteTimeout(args.Timeout, p.waitLockingTimeout)
	expireSeconds := providers.ExecuteTimeout(args.Expire, DefaultLockExpiration)
	if args.Expire != nil && math.IsNaN(*args.Expire) {
		expireSeconds = DefaultLockExpiration
	}

	ownerID := "execute:" + providers.RandomUUID()

	cancelled := &atomic.Bool{}
	acquired := make(chan error, 1)
	go func() {
		acquired <- p.acquire(ctx, keys, &ownerID, expireSeconds, true, cancelled)
	}()
	timer := time.NewTimer(providers.SecondsDuration(timeoutSeconds))
	select {
	case err := <-acquired:
		timer.Stop()
		if err != nil {
			return nil, err
		}
	case <-timer.C:
		cancelled.Store(true)
		// the acquire loop can be mid-SET: hand the keys back
		go func() {
			if err := <-acquired; err == nil {
				_, _ = p.Release(context.Background(), keys, lockingtypes.ReleaseArgs{OwnerID: &ownerID})
			}
		}()
		return nil, conflict("Timed-out acquiring lock.")
	}

	jobCtx, abort := context.WithCancelCause(ctx)
	stop := p.renewUntilStopped(keys, ownerID, expireSeconds, abort)
	defer func() {
		stop()
		abort(nil)
		_, _ = p.Release(context.Background(), keys, lockingtypes.ReleaseArgs{OwnerID: &ownerID})
	}()
	result, err := job(jobCtx)
	if err != nil {
		return nil, err
	}
	// the job ran without exclusivity when the lease was lost
	if cause := context.Cause(jobCtx); cause != nil && jobCtx.Err() != nil {
		return nil, cause
	}
	return result, nil
}

// renewUntilStopped mirrors renewUntilStopped: extendLock every third of the
// lease (at least 250ms); a lost key aborts the job.
func (p *RedisLockingProvider) renewUntilStopped(keys []string, ownerID string, expireSeconds float64, abort context.CancelCauseFunc) func() {
	interval := time.Duration(math.Max(math.Floor(expireSeconds*1000/3), 250)) * time.Millisecond
	var mu sync.Mutex
	stopped := false
	var timer *time.Timer
	var renew func()
	schedule := func() {
		mu.Lock()
		defer mu.Unlock()
		if !stopped {
			timer = time.AfterFunc(interval, renew)
		}
	}
	renew = func() {
		extended := make([]int64, len(keys))
		var firstErr error
		for i, key := range keys {
			n, err := extendLockScript.Run(context.Background(), p.client, []string{p.getKeyName(key)}, ownerID, jsNum(expireSeconds)).Int64()
			if err != nil && firstErr == nil {
				firstErr = err
			}
			extended[i] = n
		}
		mu.Lock()
		isStopped := stopped
		mu.Unlock()
		if firstErr != nil {
			if !isStopped {
				p.logger.Warn(fmt.Sprintf(`Failed to renew lock for key(s) "%s": %s. Retrying in %dms.`, strings.Join(keys, `", "`), firstErr.Error(), interval.Milliseconds()))
				schedule()
			}
			return
		}
		if isStopped {
			return
		}
		var lost []string
		for i, key := range keys {
			if extended[i] != 1 {
				lost = append(lost, key)
			}
		}
		if len(lost) == 0 {
			schedule()
			return
		}
		message := fmt.Sprintf(`Lost the lock for key(s) "%s" while the job was still running: the lease expired before it could be renewed. Increase "expire" if the job is expected to run this long.`, strings.Join(lost, `", "`))
		p.logger.Warn(message)
		abort(conflict(message))
	}
	schedule()
	return func() {
		mu.Lock()
		defer mu.Unlock()
		stopped = true
		if timer != nil {
			timer.Stop()
		}
	}
}

// Acquire mirrors acquire.
func (p *RedisLockingProvider) Acquire(ctx context.Context, keys []string, args lockingtypes.AcquireArgs) error {
	return p.acquire(ctx, keys, args.OwnerID, args.Expire, args.AwaitQueue, nil)
}

func jsNum(n float64) string { return strconv.FormatFloat(n, 'f', -1, 64) }

// acquire mirrors acquire_: every key is taken concurrently with the
// acquireLock command (owner "*" when none); with awaitQueue the attempts
// are retried with exponential backoff and jitter.
func (p *RedisLockingProvider) acquire(ctx context.Context, keys []string, ownerID *string, expire float64, awaitQueue bool, cancelled *atomic.Bool) error {
	timeoutSeconds := p.waitLockingTimeout
	if expire != 0 {
		timeoutSeconds = expire
	}
	timeoutSeconds = math.Max(timeoutSeconds, 1)
	if math.IsNaN(timeoutSeconds) {
		timeoutSeconds = 1
	}
	ttl := 0.0
	if expire != 0 {
		ttl = timeoutSeconds
	}
	owner := "*"
	if ownerID != nil {
		owner = *ownerID
	}
	var retryMu sync.Mutex
	retryDelay := p.defaultRetryInterval

	errs := make([]error, len(keys))
	var wg sync.WaitGroup
	for i, key := range keys {
		wg.Add(1)
		go func(i int, key string) {
			defer wg.Done()
			errMessage := fmt.Sprintf(`Failed to acquire lock for key "%s"`, key)
			keyName := p.getKeyName(key)
			for {
				if cancelled != nil && cancelled.Load() {
					errs[i] = conflict(errMessage)
					return
				}
				res, err := acquireLockScript.Run(context.Background(), p.client, []string{keyName}, owner, jsNum(ttl), strconv.FormatBool(awaitQueue)).Int64()
				if err != nil {
					errs[i] = err
					return
				}
				if res == 1 {
					return
				}
				if !awaitQueue {
					errs[i] = conflict(errMessage)
					return
				}
				retryMu.Lock()
				jittered := retryDelay * (0.5 + rand.Float64()*0.5)
				retryDelay = math.Min(retryDelay*p.backoffFactor, p.maximumRetryInterval)
				retryMu.Unlock()
				time.Sleep(time.Duration(jittered * float64(time.Millisecond)))
			}
		}(i, key)
	}
	wg.Wait()
	// promiseAll: every key settles, the first rejection is thrown
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// Release mirrors release: true when every key was held by the owner ("*"
// when none) and deleted.
func (p *RedisLockingProvider) Release(ctx context.Context, keys []string, args lockingtypes.ReleaseArgs) (bool, error) {
	owner := "*"
	if args.OwnerID != nil {
		owner = *args.OwnerID
	}
	all := true
	var firstErr error
	for _, key := range keys {
		n, err := releaseLockScript.Run(context.Background(), p.client, []string{p.getKeyName(key)}, owner).Int64()
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if n != 1 {
			all = false
		}
	}
	if firstErr != nil {
		return false, firstErr
	}
	return all, nil
}

// ReleaseAll mirrors releaseAll: SCAN the namespace and unlink the keys
// held by the owner ("*" when none).
func (p *RedisLockingProvider) ReleaseAll(ctx context.Context, args lockingtypes.ReleaseArgs) error {
	owner := "*"
	if args.OwnerID != nil {
		owner = *args.OwnerID
	}
	pattern := p.keyNamePrefix + "*"
	var cursor uint64
	for {
		// ioredis does not prefix SCAN patterns; GET/UNLINK are prefixed
		keys, next, err := p.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		cursor = next
		if len(keys) > 0 {
			pipe := p.client.Pipeline()
			gets := make([]*redis.StringCmd, len(keys))
			for i, key := range keys {
				gets[i] = pipe.Get(ctx, p.client.KeyPrefix+key)
			}
			_, _ = pipe.Exec(ctx)
			del := p.client.Pipeline()
			queued := 0
			for i, key := range keys {
				if v, err := gets[i].Result(); err == nil && v == owner {
					del.Unlink(ctx, p.client.KeyPrefix+key)
					queued++
				}
			}
			if queued > 0 {
				if _, err := del.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
					return err
				}
			}
		}
		if cursor == 0 {
			return nil
		}
	}
}

// Shutdown mirrors shutdown: the client disconnects.
func (p *RedisLockingProvider) Shutdown() error {
	return p.client.Close()
}
