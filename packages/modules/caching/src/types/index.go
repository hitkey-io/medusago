// Package types mirrors @medusajs/caching/src/types.
package types

import (
	"context"
)

// Registration names (CachingDefaultProvider, CachingProviderRegistrationPrefix).
const (
	CachingDefaultProvider             = "default_provider"
	CachingIdentifiersRegistrationName = "caching_providers_identifier"
	CachingProviderRegistrationPrefix  = "lp_"
)

// ProviderSetOptions are the arguments of a provider's set().
type ProviderSetOptions struct {
	Key  string
	Data any
	// TTL in seconds (nil: the provider default).
	TTL  *int
	Tags []string
	// Options mirrors options ({ autoInvalidate }); HasOptions reports an
	// options object being given (useCache always gives one).
	HasOptions     bool
	AutoInvalidate *bool
}

// ProviderClearOptions are the arguments of a provider's clear(): Options
// is false for an explicit call (every matching entry is cleared).
type ProviderClearOptions struct {
	Key            string
	Tags           []string
	Options        bool
	AutoInvalidate *bool
}

// ICachingProviderService mirrors ICachingProviderService.
type ICachingProviderService interface {
	Get(ctx context.Context, key string, tags []string) (any, error)
	Set(ctx context.Context, opts ProviderSetOptions) error
	Clear(ctx context.Context, opts ProviderClearOptions) error
}

// Shutdowner is a provider with a shutdown().
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// MemoryCacheModuleOptions mirrors MemoryCacheModuleOptions.
type MemoryCacheModuleOptions struct {
	TTL         *int
	MaxKeys     *int
	CheckPeriod *int
	UseClones   *bool
	MaxSize     *int
}

// CachingModuleOptions mirrors CachingModuleOptions (ttl, in_memory).
type CachingModuleOptions struct {
	TTL             *int
	InMemoryEnabled bool
	InMemory        MemoryCacheModuleOptions
}

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

// OptionsFromMap reads the module options.
func OptionsFromMap(m map[string]any) CachingModuleOptions {
	var o CachingModuleOptions
	o.TTL = intOpt(m, "ttl")
	if im, ok := m["in_memory"].(map[string]any); ok {
		o.InMemoryEnabled, _ = im["enable"].(bool)
		o.InMemory.TTL = intOpt(im, "ttl")
		o.InMemory.MaxKeys = intOpt(im, "maxKeys")
		o.InMemory.CheckPeriod = intOpt(im, "checkPeriod")
		o.InMemory.MaxSize = intOpt(im, "maxSize")
		if b, ok := im["useClones"].(bool); ok {
			o.InMemory.UseClones = &b
		}
	}
	return o
}
