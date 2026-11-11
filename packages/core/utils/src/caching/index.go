// Package caching mirrors @medusajs/utils/src/caching: useCache and the
// options of the @Cached decorator (query.graph / query.index cache
// options).
package caching

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Module is the caching module service (ICachingModuleService).
type Module interface {
	Get(ctx context.Context, opts GetOptions) (any, error)
	Set(ctx context.Context, opts SetOptions) error
	Clear(ctx context.Context, opts ClearOptions) error
	ComputeKey(input any) (string, error)
	ComputeTags(input any, opts *ComputeTagsOptions) ([]string, error)
}

// GetOptions are the options of get().
type GetOptions struct {
	Key       string
	Tags      []string
	Providers []string
}

// ProviderTTL is a provider given as { id, ttl }.
type ProviderTTL struct {
	ID  string
	TTL *int
}

// SetOptions are the options of set().
type SetOptions struct {
	Key                  string
	Data                 any
	TTL                  *int
	Tags                 []string
	ComputeAutomaticTags bool
	Providers            []ProviderTTL
	// HasOptions reports an options object being given ({ autoInvalidate }
	// from useCache); AutoInvalidate is nil when its value is undefined.
	HasOptions     bool
	AutoInvalidate *bool
}

// ClearOptions are the options of clear(). Options is false for an
// explicit call without options (every matching entry is cleared).
type ClearOptions struct {
	Key            string
	Tags           []string
	Options        bool
	AutoInvalidate *bool
	Providers      []string
}

// EntityReference mirrors EntityReference.
type EntityReference struct {
	Type      string
	ID        any
	IsInArray bool
}

// ComputeTagsOptions are the options of computeTags().
type ComputeTagsOptions struct {
	Entities  []EntityReference
	Operation string
}

// CacheOptions mirrors the `cache` option of query.graph / query.index
// (RemoteJoinerOptions.cache).
type CacheOptions struct {
	// Enable is nil when not given.
	Enable               *bool
	Key                  string
	Tags                 []string
	ComputeAutomaticTags bool
	TTL                  *int
	AutoInvalidate       *bool
	Providers            []string
}

// Enabled returns the cache option { enable: true }.
func Enabled() *CacheOptions {
	t := true
	return &CacheOptions{Enable: &t}
}

// WithTags returns the cache option { tags, computeAutomaticTags }.
func WithTags(computeAutomaticTags bool, tags ...string) *CacheOptions {
	return &CacheOptions{Tags: tags, ComputeAutomaticTags: computeAutomaticTags}
}

// IsEnabled mirrors isCacheEnabled (query/src/query/cache.ts).
func (o *CacheOptions) IsEnabled() bool {
	if o == nil {
		return false
	}
	if o.Enable != nil && !*o.Enable {
		return false
	}
	return (o.Enable != nil && *o.Enable) || o.Key != "" || o.TTL != nil || o.Tags != nil || o.ComputeAutomaticTags ||
		(o.AutoInvalidate != nil && *o.AutoInvalidate) || len(o.Providers) > 0
}

// UseCacheOptions are the options of useCache.
type UseCacheOptions struct {
	Enable bool
	// Key is the cache key; KeyInput is hashed with computeKey when Key is
	// empty.
	Key                  string
	KeyInput             any
	Tags                 []string
	ComputeAutomaticTags bool
	TTL                  *int
	AutoInvalidate       *bool
	Providers            []string
	// Module is the caching module (nil: not loaded); FlagEnabled the
	// caching feature flag.
	Module      Module
	FlagEnabled bool
	Logger      *slog.Logger
}

// UseCache mirrors useCache: the cached data (JSON) when present, else the
// callback's result, stored in the background.
//
// The value given to set() is snapshotted (JSON) before UseCache returns:
// upstream stringifies it a few ticks later, before the caller's next I/O.
func UseCache(ctx context.Context, cb func() (any, error), opts UseCacheOptions) (any, error) {
	if !opts.Enable || !opts.FlagEnabled || opts.Module == nil {
		return cb()
	}
	key := opts.Key
	if key == "" {
		k, err := opts.Module.ComputeKey(opts.KeyInput)
		if err != nil {
			return nil, err
		}
		key = k
	}
	data, err := opts.Module.Get(ctx, GetOptions{Key: key, Tags: opts.Tags, Providers: opts.Providers})
	if err != nil {
		return nil, err
	}
	if Truthy(data) {
		return data, nil
	}
	result, err := cb()
	if err != nil {
		return nil, err
	}
	snapshot, err := Snapshot(result)
	if err != nil {
		return result, nil
	}
	providers := make([]ProviderTTL, 0, len(opts.Providers))
	for _, p := range opts.Providers {
		providers = append(providers, ProviderTTL{ID: p})
	}
	if len(providers) == 0 {
		providers = nil
	}
	set := SetOptions{Key: key, Data: snapshot, TTL: opts.TTL, Tags: opts.Tags, ComputeAutomaticTags: opts.ComputeAutomaticTags,
		Providers: providers, HasOptions: true, AutoInvalidate: opts.AutoInvalidate}
	go func() {
		if err := opts.Module.Set(context.WithoutCancel(ctx), set); err != nil {
			logger := opts.Logger
			if logger == nil {
				logger = slog.Default()
			}
			logger.Error(fmt.Sprintf("An error occurred while setting cache for key: %s\n%s", key, err.Error()))
		}
	}()
	return result, nil
}

// Snapshot returns the JSON value of v (JSON.parse(JSON.stringify(v))):
// objects as *dal.Record, numbers as float64.
func Snapshot(v any) (any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return dal.DecodeOrderedJSON(b)
}

// JSONStringify mirrors JSON.stringify (no HTML escaping).
func JSONStringify(v any) (string, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// Truthy mirrors JavaScript truthiness for a cached value.
func Truthy(v any) bool {
	switch t := v.(type) {
	case nil:
		return false
	case bool:
		return t
	case string:
		return t != ""
	case float64:
		return t != 0 && t == t
	}
	return true
}

// StableStringify mirrors fast-json-stable-stringify: JSON with the object
// keys sorted, undefined values left out.
func StableStringify(v any) (string, error) {
	snap, err := Snapshot(v)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	if err := writeStable(&b, snap); err != nil {
		return "", err
	}
	return b.String(), nil
}

func writeStable(b *bytes.Buffer, v any) error {
	switch t := v.(type) {
	case *dal.Record:
		keys := append([]string{}, t.Keys()...)
		sort.Strings(keys)
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			writeString(b, k)
			b.WriteByte(':')
			val, _ := t.Get(k)
			if err := writeStable(b, val); err != nil {
				return err
			}
		}
		b.WriteByte('}')
	case []any:
		b.WriteByte('[')
		for i, x := range t {
			if i > 0 {
				b.WriteByte(',')
			}
			if err := writeStable(b, x); err != nil {
				return err
			}
		}
		b.WriteByte(']')
	case string:
		writeString(b, t)
	default:
		out, err := json.Marshal(t)
		if err != nil {
			return err
		}
		b.Write(out)
	}
	return nil
}

// writeString writes a JSON string like JSON.stringify (no HTML escaping,
// U+2028 / U+2029 kept as is).
func writeString(b *bytes.Buffer, s string) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	out := strings.TrimRight(buf.String(), "\n")
	out = strings.ReplaceAll(out, ` `, " ")
	out = strings.ReplaceAll(out, ` `, " ")
	b.WriteString(out)
}
