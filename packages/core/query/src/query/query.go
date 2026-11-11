package query

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	typesjoiner "github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// Query mirrors Query: the cross-module read layer.
type Query struct {
	configs []*typesjoiner.ModuleJoinerConfig
	catalog *joiner.GraphCatalog
	fetcher *ModuleDataFetcher
	// indexModule is the index module (nil when it is not loaded).
	indexModule IndexService
	// searchModule is the search module (nil when it is not loaded).
	searchModule SearchService
	// translations mirrors the translation feature flag (applyTranslations).
	translations bool
	// caching is the caching module (nil when it is not loaded) and
	// cachingFlag the caching feature flag (@Cached).
	caching     caching.Module
	cachingFlag bool
}

// New builds Query over the joiner configs of the loaded modules (link
// modules included), like createQuery.
func New(configs []*typesjoiner.ModuleJoinerConfig, relationMap joiner.RelationMap, services map[string]ModuleService) (*Query, error) {
	catalog, err := joiner.NewGraphCatalog(configs, relationMap)
	if err != nil {
		return nil, err
	}
	return &Query{configs: configs, catalog: catalog, fetcher: &ModuleDataFetcher{Services: services}}, nil
}

// GraphResult mirrors GraphResultSet.
type GraphResult struct {
	Data     []any
	Metadata *dal.Record
}

// MarshalJSON writes { data, metadata? } like the upstream result.
func (r *GraphResult) MarshalJSON() ([]byte, error) {
	out := dal.NewRecord()
	out.Set("data", r.Data)
	if r.Metadata != nil {
		out.Set("metadata", r.Metadata)
	}
	return out.MarshalJSON()
}

// Graph mirrors query.graph (cached with the cache option, see cache.go).
func (q *Query) Graph(ctx context.Context, in GraphInput, opts joiner.Options) (*GraphResult, error) {
	return q.cached(ctx, in, nil, opts, func() (*GraphResult, error) {
		return q.run(ctx, ToRemoteQuery(in, q.configs), opts)
	})
}

// RemoteQuery mirrors the legacy remoteQuery(remoteQueryObjectFromString(...))
// call: the result carries metadata when the query is paginated (rows +
// metadata upstream), otherwise only the data.
func (q *Query) RemoteQuery(ctx context.Context, obj *dal.Record, opts ...joiner.Options) (*GraphResult, error) {
	var o joiner.Options
	if len(opts) > 0 {
		o = opts[0]
	}
	return q.run(ctx, obj, o)
}

func (q *Query) run(ctx context.Context, remote *dal.Record, opts joiner.Options) (*GraphResult, error) {
	jq := ToRemoteJoinerQuery(remote)
	serviceConfig := q.catalog.GetServiceConfig(jq.Service, jq.Alias, "")
	if serviceConfig == nil {
		return nil, fmt.Errorf(`Service with alias "%s" was not found.`, jq.Alias)
	}
	plan, err := joiner.CompileQuery(jq, serviceConfig, q.catalog)
	if err != nil {
		return nil, err
	}
	res, err := joiner.ExecutePlan(ctx, plan, q.fetcher, q.catalog, opts)
	if err != nil {
		return nil, err
	}
	out := &GraphResult{Data: res.Data}
	if res.Paged {
		out.Metadata = res.Metadata
	}
	if out.Data == nil {
		out.Data = []any{}
	}
	if opts.Locale != "" {
		if err := q.ApplyTranslations(ctx, opts.Locale, out.Data); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// SuffixLister is a module service Query reads from: the generic
// MedusaService or a module service overriding some list methods.
type SuffixLister interface {
	ListBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error)
	ListAndCountBySuffix(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error)
}

// MedusaServiceAdapter exposes a module service to Query.
type MedusaServiceAdapter struct {
	Service SuffixLister
}

// List calls list<suffix>.
func (a MedusaServiceAdapter) List(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	return a.Service.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCount calls listAndCount<suffix>.
func (a MedusaServiceAdapter) ListAndCount(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	return a.Service.ListAndCountBySuffix(ctx, suffix, filters, config)
}

// IndexQueryInput is what query.index() asks the index module for
// (IndexTypes.IndexQueryConfig).
type IndexQueryInput struct {
	Fields      []string
	Filters     *dal.Record
	JoinFilters *dal.Record
	Pagination  *dal.Record
	IdsOnly     bool
}

// IndexService is the part of the index module query.index() uses.
type IndexService interface {
	QueryIndex(ctx context.Context, in IndexQueryInput) (data []any, metadata *dal.Record, err error)
}

// SetIndexModule registers the index module (createQuery's indexModule).
func (q *Query) SetIndexModule(index IndexService) { q.indexModule = index }

// Index mirrors query.index(): the index module filters, orders and pages
// the entity ids, then query.graph loads the requested fields of those
// ids, in the index order.
func (q *Query) Index(ctx context.Context, in GraphInput, joinFilters *dal.Record, opts joiner.Options) (*GraphResult, error) {
	return q.cached(ctx, in, joinFilters, opts, func() (*GraphResult, error) {
		return q.index(ctx, in, joinFilters, opts)
	})
}

func (q *Query) index(ctx context.Context, in GraphInput, joinFilters *dal.Record, opts joiner.Options) (*GraphResult, error) {
	if q.indexModule == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Index module is not loaded.")
	}
	mainEntity := in.Entity
	fields := []string{mainEntity + ".id"}
	filters := dal.NewRecord()
	if in.Filters != nil {
		filters.Set(mainEntity, in.Filters)
	}
	jf := dal.NewRecord()
	if joinFilters != nil {
		jf.Set(mainEntity, joinFilters)
	}
	var pagination *dal.Record
	if in.Pagination != nil {
		pagination = dal.NewRecord()
		for _, k := range in.Pagination.Keys() {
			v, _ := in.Pagination.Get(k)
			if k == "order" {
				if o, ok := v.(*dal.Record); ok {
					wrapped := dal.NewRecord()
					wrapped.Set(mainEntity, unflattenObjectKeys(o))
					v = wrapped
				}
			}
			pagination.Set(k, v)
		}
	}
	data, metadata, err := q.indexModule.QueryIndex(ctx, IndexQueryInput{Fields: fields, Filters: filters, JoinFilters: jf, Pagination: pagination, IdsOnly: true})
	if err != nil {
		return nil, err
	}

	ids := make([]any, 0, len(data))
	for _, d := range data {
		if r, ok := d.(*dal.Record); ok {
			id, _ := r.Get("id")
			ids = append(ids, id)
		}
	}
	idFilters := dal.NewRecord()
	idFilters.Set("id", ids)
	var take any = float64(len(data))
	if in.Pagination != nil {
		if t, ok := in.Pagination.Get("take"); ok && t != nil {
			take = t
		}
	}
	graphPagination := dal.NewRecord()
	graphPagination.Set("take", take)
	graphIn := in
	graphIn.Filters = idFilters
	graphIn.Pagination = graphPagination

	final := data
	if len(data) > 0 {
		res, err := q.Graph(ctx, graphIn, opts)
		if err != nil {
			return nil, err
		}
		final = mergeInitialData(res.Data, data)
	}
	if final == nil {
		final = []any{}
	}
	// index() translates the final result set for the request locale too
	// (graph() already did for the fetched entities)
	if opts.Locale != "" {
		if err := q.ApplyTranslations(ctx, opts.Locale, final); err != nil {
			return nil, err
		}
	}
	return &GraphResult{Data: final, Metadata: metadata}, nil
}

// mergeInitialData mirrors the root level of the joiner's
// mergeInitialData: the fetched items in the order of the initial data
// (the index ids), the initial entry itself when the item was not found,
// and the initial properties assigned over the item.
func mergeInitialData(items []any, initialData []any) []any {
	byID := map[string]*dal.Record{}
	for _, it := range items {
		if r, ok := it.(*dal.Record); ok {
			id, _ := r.Get("id")
			byID[fmt.Sprint(id)] = r
		}
	}
	out := make([]any, len(initialData))
	for i, ini := range initialData {
		iniRec, _ := ini.(*dal.Record)
		var id any
		if iniRec != nil {
			id, _ = iniRec.Get("id")
		}
		item := byID[fmt.Sprint(id)]
		if item == nil {
			out[i] = ini
			continue
		}
		if iniRec != nil {
			for _, k := range iniRec.Keys() {
				v, _ := iniRec.Get(k)
				item.Set(k, v)
			}
		}
		out[i] = item
	}
	return out
}

// unflattenObjectKeys mirrors unflattenObjectKeys for an order object.
func unflattenObjectKeys(flattened *dal.Record) *dal.Record {
	result := dal.NewRecord()
	for _, key := range flattened.Keys() {
		if strings.Contains(key, ".") {
			continue
		}
		v, _ := flattened.Get(key)
		result.Set(key, v)
	}
	for _, key := range flattened.Keys() {
		if !strings.Contains(key, ".") {
			continue
		}
		value, _ := flattened.Get(key)
		parts := strings.Split(key, ".")
		current := result
		for i, part := range parts {
			if i == len(parts)-1 {
				current.Set(part, value)
				continue
			}
			v, _ := current.Get(part)
			next, ok := v.(*dal.Record)
			if !ok {
				next = dal.NewRecord()
				current.Set(part, next)
			}
			current = next
		}
	}
	return result
}
