package query

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// SetCaching registers the caching module (resolved from the container by
// @Cached) and the caching feature flag.
func (q *Query) SetCaching(module caching.Module, flagEnabled bool) {
	q.caching = module
	q.cachingFlag = flagEnabled
}

// queryOptionsRecord is the queryOptions argument of graph() / index() as a
// JavaScript object (the cache key input).
func queryOptionsRecord(in GraphInput, joinFilters *dal.Record) *dal.Record {
	r := dal.NewRecord()
	r.Set("entity", in.Entity)
	if in.Fields != nil {
		fields := make([]any, len(in.Fields))
		for i, f := range in.Fields {
			fields[i] = f
		}
		r.Set("fields", fields)
	}
	if in.Filters != nil {
		r.Set("filters", in.Filters)
	}
	if joinFilters != nil {
		r.Set("joinFilters", joinFilters)
	}
	if in.Pagination != nil {
		r.Set("pagination", in.Pagination)
	}
	if in.Context != nil {
		r.Set("context", in.Context)
	}
	if in.WithDeleted {
		r.Set("withDeleted", true)
	}
	return r
}

// joinerOptionsRecord is the options argument without initialData and
// cache.
func joinerOptionsRecord(opts joiner.Options) *dal.Record {
	r := dal.NewRecord()
	if opts.ThrowIfKeyNotFound {
		r.Set("throwIfKeyNotFound", true)
	}
	if opts.ThrowIfRelationNotFound {
		r.Set("throwIfRelationNotFound", true)
	}
	if opts.Locale != "" {
		r.Set("locale", opts.Locale)
	}
	return r
}

// cached mirrors @Cached(queryCacheDecoratorOptions) on graph() and
// index(): with an enabled cache option, the result is read from and
// stored in the caching module under the key given, or the hash of
// { queryOptions, options }.
func (q *Query) cached(ctx context.Context, in GraphInput, joinFilters *dal.Record, opts joiner.Options, run func() (*GraphResult, error)) (*GraphResult, error) {
	c := opts.Cache
	if q.caching == nil || !q.cachingFlag || !c.IsEnabled() {
		return run()
	}
	keyInput := dal.NewRecord()
	keyInput.Set("queryOptions", queryOptionsRecord(in, joinFilters))
	keyInput.Set("options", joinerOptionsRecord(opts))
	v, err := caching.UseCache(ctx, func() (any, error) { return run() }, caching.UseCacheOptions{
		Enable: true, Key: c.Key, KeyInput: keyInput, Tags: c.Tags, ComputeAutomaticTags: c.ComputeAutomaticTags,
		TTL: c.TTL, AutoInvalidate: c.AutoInvalidate, Providers: c.Providers, Module: q.caching, FlagEnabled: q.cachingFlag,
	})
	if err != nil {
		return nil, err
	}
	switch t := v.(type) {
	case *GraphResult:
		return t, nil
	case *dal.Record:
		// the cached JSON { data, metadata }
		out := &GraphResult{Data: []any{}}
		if d, ok := t.Get("data"); ok {
			if list, ok := d.([]any); ok {
				out.Data = list
			}
		}
		if md, ok := t.Get("metadata"); ok {
			out.Metadata, _ = md.(*dal.Record)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unexpected cached query result %T", v)
}
