package query

import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// SearchService is the part of the Search Module query.search() uses.
// Queries and results are plain objects (SearchQuery / SearchResult).
type SearchService interface {
	SearchMany(ctx context.Context, queries []*dal.Record) ([]*dal.Record, error)
	ListRetrievableFields(index string) ([]string, error)
	IndexPrimaryKey(index string) (string, error)
}

// SetSearchModule registers the search module (createQuery's searchModule).
func (q *Query) SetSearchModule(search SearchService) { q.searchModule = search }

// SearchResultSet mirrors SearchResultSet: the (hydrated) documents and the
// engine's result.
type SearchResultSet struct {
	Data         []any
	SearchResult *dal.Record
}

type preparedSearch struct {
	entity      string
	engineQuery *dal.Record
	graphFields []string
	context     *dal.Record
}

// Search mirrors query.search(queries): one provider round-trip for the
// queries, then the fields the index cannot return are fetched by id
// through query.graph, keeping the search order.
func (q *Query) Search(ctx context.Context, queries []*dal.Record, opts joiner.Options) ([]*SearchResultSet, error) {
	if q.searchModule == nil {
		return nil, common.NewMedusaError(common.MedusaErrorTypes.INVALID_DATA, "Search module is not loaded.")
	}
	prepared := make([]preparedSearch, len(queries))
	engineQueries := make([]*dal.Record, len(queries))
	for i, query := range queries {
		p, err := q.prepareSearchQuery(query)
		if err != nil {
			return nil, err
		}
		prepared[i] = p
		engineQueries[i] = p.engineQuery
	}
	searchResults, err := q.searchModule.SearchMany(ctx, engineQueries)
	if err != nil {
		return nil, err
	}
	out := make([]*SearchResultSet, len(searchResults))
	for i, result := range searchResults {
		set, err := q.hydrateSearchResult(ctx, prepared[i], result, opts)
		if err != nil {
			return nil, err
		}
		out[i] = set
	}
	return out, nil
}

func (q *Query) prepareSearchQuery(query *dal.Record) (preparedSearch, error) {
	entityValue, _ := query.Get("entity")
	entity, _ := entityValue.(string)
	retrievableFields, err := q.searchModule.ListRetrievableFields(entity)
	if err != nil {
		return preparedSearch{}, err
	}
	requested := retrievableFields
	if v, ok := query.Get("fields"); ok && v != nil {
		requested = toStringList(v)
	}
	retrievable := map[string]bool{}
	for _, f := range retrievableFields {
		retrievable[f] = true
	}
	engineQuery := dal.NewRecord()
	var queryContext *dal.Record
	for _, k := range query.Keys() {
		v, _ := query.Get(k)
		if k == "context" {
			queryContext, _ = v.(*dal.Record)
			continue
		}
		engineQuery.Set(k, v)
	}
	engineFields := []any{}
	var graphFields []string
	for _, f := range requested {
		if retrievable[f] {
			engineFields = append(engineFields, f)
		} else {
			graphFields = append(graphFields, f)
		}
	}
	engineQuery.Set("fields", engineFields)
	return preparedSearch{entity: entity, engineQuery: engineQuery, graphFields: graphFields, context: queryContext}, nil
}

func toStringList(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			s, _ := e.(string)
			out = append(out, s)
		}
		return out
	}
	return nil
}

func (q *Query) hydrateSearchResult(ctx context.Context, p preparedSearch, result *dal.Record, opts joiner.Options) (*SearchResultSet, error) {
	hitsValue, _ := result.Get("hits")
	hits, _ := hitsValue.([]any)
	data := make([]any, len(hits))
	ids := make([]any, len(hits))
	for i, h := range hits {
		hit, _ := h.(*dal.Record)
		if hit == nil {
			continue
		}
		data[i], _ = hit.Get("document")
		ids[i], _ = hit.Get("id")
	}
	if len(p.graphFields) > 0 && len(data) > 0 {
		primaryKey, err := q.searchModule.IndexPrimaryKey(p.entity)
		if err != nil {
			return nil, err
		}
		hydrationFields := p.graphFields
		hasKey := false
		for _, f := range hydrationFields {
			if f == primaryKey {
				hasKey = true
			}
		}
		if !hasKey {
			hydrationFields = append(append([]string{}, hydrationFields...), primaryKey)
		}
		filters := dal.NewRecord()
		filters.Set(primaryKey, ids)
		pagination := dal.NewRecord()
		pagination.Set("take", float64(len(hits)))
		expanded, err := q.Graph(ctx, GraphInput{Entity: p.entity, Fields: hydrationFields, Filters: filters, Pagination: pagination, Context: p.context}, opts)
		if err != nil {
			return nil, err
		}
		data = mergeInitialData(expanded.Data, data)
	}
	return &SearchResultSet{Data: data, SearchResult: result}, nil
}
