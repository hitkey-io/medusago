// Package services mirrors @medusajs/caching-redis/src/services.
package services

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/redis/go-redis/v9"

	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	"github.com/hitkey-io/medusago/packages/modules/caching/src/types"
)

// Identifier mirrors RedisCachingProvider.identifier.
const Identifier = "cache-redis"

// RedisCachingProvider mirrors RedisCachingProvider: entries are hashes
// <prefix><key> { data, options } (data: a format byte, 0 plain or 1 gzip,
// then the JSON), with a forward tag index <prefix>tag:<hashed tag> (set of
// entry keys) and a reverse index <prefix>tags:<key> (set of hashed tags).
type RedisCachingProvider struct {
	client               redis.UniversalClient
	keyNamePrefix        string
	defaultTTL           int
	compressionThreshold int
	hasher               func(string) string
	logger               *slog.Logger
}

// Options are the provider options (ttl, compressionThreshold).
type Options struct {
	TTL                  *int
	CompressionThreshold *int
}

// NewRedisCachingProvider mirrors the constructor (ttl 3600, compression
// threshold 2048).
func NewRedisCachingProvider(client redis.UniversalClient, prefix string, hasher func(string) string, logger *slog.Logger, opts Options) *RedisCachingProvider {
	if logger == nil {
		logger = slog.Default()
	}
	p := &RedisCachingProvider{client: client, keyNamePrefix: prefix, defaultTTL: 3600, compressionThreshold: 2048, hasher: hasher, logger: logger}
	if opts.TTL != nil {
		p.defaultTTL = *opts.TTL
	}
	if opts.CompressionThreshold != nil {
		p.compressionThreshold = *opts.CompressionThreshold
	}
	return p
}

// Client returns the Redis client.
func (p *RedisCachingProvider) Client() redis.UniversalClient { return p.client }

// Shutdown closes the connection.
func (p *RedisCachingProvider) Shutdown(context.Context) error { return p.client.Close() }

// isConnectionError mirrors isConnectionError.
func isConnectionError(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return true
	}
	msg := err.Error()
	for _, s := range []string{"ECONNREFUSED", "ENOTFOUND", "ETIMEDOUT", "ECONNRESET", "EPIPE", "connection refused", "client is closed",
		"Connection is closed", "i/o timeout", "broken pipe", "connection reset"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

func (p *RedisCachingProvider) keyName(key string) string { return p.keyNamePrefix + key }

// tagKey is the forward index of a tag (hashed unless already).
func (p *RedisCachingProvider) tagKey(tag string, isHashed bool) string {
	if !isHashed {
		tag = p.hasher(tag)
	}
	return p.keyNamePrefix + "tag:" + tag
}

// tagsKey is the reverse index of an entry.
func (p *RedisCachingProvider) tagsKey(key string) string { return p.keyNamePrefix + "tags:" + key }

// logicalKey mirrors keyName.replace(prefix, "").
func (p *RedisCachingProvider) logicalKey(keyName string) string {
	return strings.Replace(keyName, p.keyNamePrefix, "", 1)
}

// removeTagIndexes mirrors #removeTagIndexes.
func (p *RedisCachingProvider) removeTagIndexes(ctx context.Context, keyNames []string) error {
	if len(keyNames) == 0 {
		return nil
	}
	tagsKeys := make([]string, len(keyNames))
	read := p.client.Pipeline()
	cmds := make([]*redis.StringSliceCmd, len(keyNames))
	for i, k := range keyNames {
		tagsKeys[i] = p.tagsKey(p.logicalKey(k))
		cmds[i] = read.SMembers(ctx, tagsKeys[i])
	}
	if _, err := read.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return err
	}
	cleanup := p.client.Pipeline()
	for i, k := range keyNames {
		hashed, _ := cmds[i].Result()
		for _, h := range hashed {
			cleanup.SRem(ctx, p.tagKey(h, true), k)
		}
		cleanup.Unlink(ctx, tagsKeys[i])
	}
	_, err := cleanup.Exec(ctx)
	return err
}

// deleteEntries mirrors #deleteEntries.
func (p *RedisCachingProvider) deleteEntries(ctx context.Context, keyNames []string) error {
	if len(keyNames) == 0 {
		return nil
	}
	if err := p.removeTagIndexes(ctx, keyNames); err != nil {
		return err
	}
	del := p.client.Pipeline()
	for _, k := range keyNames {
		del.Unlink(ctx, k)
	}
	_, err := del.Exec(ctx)
	return err
}

func secs(n int) time.Duration { return time.Duration(n) * time.Second }

// jsLength is the length of a string in UTF-16 code units (data.length).
func jsLength(s string) int { return len(utf16.Encode([]rune(s))) }

// compressData mirrors #compressData.
func (p *RedisCachingProvider) compressData(data string) []byte {
	if jsLength(data) <= p.compressionThreshold {
		return append([]byte{0}, data...)
	}
	var buf bytes.Buffer
	buf.WriteByte(1)
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(data)); err != nil {
		return append([]byte{0}, data...)
	}
	if err := zw.Close(); err != nil {
		return append([]byte{0}, data...)
	}
	return buf.Bytes()
}

// decompressData mirrors #decompressData.
func decompressData(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	switch b[0] {
	case 0:
		return string(b[1:])
	case 1:
		zr, err := gzip.NewReader(bytes.NewReader(b[1:]))
		if err != nil {
			return string(b[1:])
		}
		out, err := io.ReadAll(zr)
		if err != nil {
			return string(b[1:])
		}
		return string(out)
	}
	return string(b)
}

func (p *RedisCachingProvider) parse(b []byte) (any, error) {
	return dal.DecodeOrderedJSON([]byte(decompressData(b)))
}

// Get mirrors get({ key, tags }).
func (p *RedisCachingProvider) Get(ctx context.Context, key string, tags []string) (any, error) {
	if key != "" {
		b, err := p.client.HGet(ctx, p.keyName(key), "data").Bytes()
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		if err != nil {
			if isConnectionError(err) {
				p.logger.Warn(fmt.Sprintf("[redis-cache] Redis connection error during get operation, returning empty array to trigger fallback to original data source. Error: %s", err.Error()))
				return nil, nil
			}
			return nil, err
		}
		return p.parse(b)
	}
	if len(tags) > 0 {
		keys, err := p.keysOfTags(ctx, tags)
		if err != nil {
			if isConnectionError(err) {
				p.logger.Warn(fmt.Sprintf("[redis-cache] Redis connection error during get operation, returning empty array to trigger fallback to original data source. Error: %s", err.Error()))
				return nil, nil
			}
			return nil, err
		}
		if len(keys) == 0 {
			return []any{}, nil
		}
		pipe := p.client.Pipeline()
		cmds := make([]*redis.StringCmd, len(keys))
		for i, k := range keys {
			cmds[i] = pipe.HGet(ctx, k, "data")
		}
		_, _ = pipe.Exec(ctx)
		results := []any{}
		for _, c := range cmds {
			b, err := c.Bytes()
			if err != nil || len(b) == 0 {
				continue
			}
			v, err := p.parse(b)
			if err != nil {
				p.logger.Warn(fmt.Sprintf("[redis-cache] Skipping corrupted cache entry: %s", err.Error()))
				continue
			}
			if caching.Truthy(v) {
				results = append(results, v)
			}
		}
		return results, nil
	}
	return nil, nil
}

// keysOfTags returns the entry keys of the tags (set order of first
// appearance).
func (p *RedisCachingProvider) keysOfTags(ctx context.Context, tags []string) ([]string, error) {
	pipe := p.client.Pipeline()
	cmds := make([]*redis.StringSliceCmd, len(tags))
	for i, t := range tags {
		cmds[i] = pipe.SMembers(ctx, p.tagKey(t, false))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	var keys []string
	seen := map[string]bool{}
	for _, c := range cmds {
		members, _ := c.Result()
		for _, k := range members {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
	}
	return keys, nil
}

// Set mirrors set({ key, data, ttl, tags, options }).
func (p *RedisCachingProvider) Set(ctx context.Context, o types.ProviderSetOptions) error {
	err := p.set(ctx, o)
	if err != nil && isConnectionError(err) {
		p.logger.Warn(fmt.Sprintf("[redis-cache] Redis connection error during set operation, relying on IORedis retry mechanism. Error: %s", err.Error()))
		return nil
	}
	return err
}

func (p *RedisCachingProvider) set(ctx context.Context, o types.ProviderSetOptions) error {
	keyName := p.keyName(o.Key)
	serialized, err := caching.JSONStringify(o.Data)
	if err != nil {
		return err
	}
	ttl := p.defaultTTL
	if o.TTL != nil {
		ttl = *o.TTL
	}
	finalData := p.compressData(serialized)
	hashedTags := make([]string, len(o.Tags))
	for i, t := range o.Tags {
		hashedTags[i] = p.hasher(t)
	}
	pipe := p.client.Pipeline()
	pipe.HSet(ctx, keyName, "data", finalData)
	if o.HasOptions {
		opts := dal.NewRecord()
		if o.AutoInvalidate != nil {
			opts.Set("autoInvalidate", *o.AutoInvalidate)
		}
		b, _ := json.Marshal(opts)
		pipe.HSet(ctx, keyName, "options", string(b))
	}
	if ttl != 0 {
		pipe.Expire(ctx, keyName, secs(ttl))
	}
	if len(hashedTags) > 0 {
		tagsKey := p.tagsKey(o.Key)
		members := make([]any, len(hashedTags))
		for i, h := range hashedTags {
			members[i] = h
		}
		pipe.SAdd(ctx, tagsKey, members...)
		if ttl != 0 {
			pipe.Expire(ctx, tagsKey, secs(ttl+60))
		}
		for _, h := range hashedTags {
			tagKey := p.tagKey(h, true)
			pipe.SAdd(ctx, tagKey, keyName)
			if ttl != 0 {
				pipe.Expire(ctx, tagKey, secs(ttl+60))
			}
		}
	}
	_, err = pipe.Exec(ctx)
	return err
}

// Clear mirrors clear({ key, tags, options }).
func (p *RedisCachingProvider) Clear(ctx context.Context, o types.ProviderClearOptions) error {
	err := p.clear(ctx, o)
	if err != nil && isConnectionError(err) {
		p.logger.Warn(fmt.Sprintf("[redis-cache] Redis connection error during clear operation, relying on IORedis retry mechanism. Error: %s", err.Error()))
		return nil
	}
	return err
}

func (p *RedisCachingProvider) clear(ctx context.Context, o types.ProviderClearOptions) error {
	if o.Key != "" {
		return p.deleteEntries(ctx, []string{p.keyName(o.Key)})
	}
	if len(o.Tags) == 0 {
		return nil
	}
	for _, t := range o.Tags {
		if t == "*" {
			return p.Flush(ctx)
		}
	}
	keys, err := p.keysOfTags(ctx, o.Tags)
	if err != nil || len(keys) == 0 {
		return err
	}
	if !o.Options {
		return p.deleteEntries(ctx, keys)
	}
	if o.AutoInvalidate == nil || !*o.AutoInvalidate {
		return nil
	}
	pipe := p.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(keys))
	for i, k := range keys {
		cmds[i] = pipe.HGet(ctx, k, "options")
	}
	_, _ = pipe.Exec(ctx)
	var toDelete []string
	for i, k := range keys {
		raw, err := cmds[i].Result()
		if err != nil || raw == "" {
			toDelete = append(toDelete, k)
			continue
		}
		var entryOptions map[string]any
		if json.Unmarshal([]byte(raw), &entryOptions) != nil {
			toDelete = append(toDelete, k)
			continue
		}
		auto := true
		if v, ok := entryOptions["autoInvalidate"]; ok && v != nil {
			auto = caching.Truthy(v)
		}
		if auto {
			toDelete = append(toDelete, k)
		}
	}
	return p.deleteEntries(ctx, toDelete)
}

// Flush mirrors flush: every key with the prefix is unlinked.
func (p *RedisCachingProvider) Flush(ctx context.Context) error {
	var cursor uint64
	for {
		keys, next, err := p.client.Scan(ctx, cursor, p.keyNamePrefix+"*", 1000).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := p.client.Unlink(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			return nil
		}
	}
}
