package query

import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	typesjoiner "github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// Query mirrors Query: the cross-module read layer.
type Query struct {
	configs []*typesjoiner.ModuleJoinerConfig
	catalog *joiner.GraphCatalog
	fetcher *ModuleDataFetcher
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

// Graph mirrors query.graph.
func (q *Query) Graph(ctx context.Context, in GraphInput, opts joiner.Options) (*GraphResult, error) {
	return q.run(ctx, ToRemoteQuery(in, q.configs), opts)
}

// RemoteQuery mirrors the legacy remoteQuery(remoteQueryObjectFromString(...))
// call: the result carries metadata when the query is paginated (rows +
// metadata upstream), otherwise only the data.
func (q *Query) RemoteQuery(ctx context.Context, obj *dal.Record) (*GraphResult, error) {
	return q.run(ctx, obj, joiner.Options{})
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
	return out, nil
}

// MedusaServiceAdapter exposes a module service to Query.
type MedusaServiceAdapter struct {
	Service *utilsmodulessdk.MedusaService
}

// List calls list<suffix>.
func (a MedusaServiceAdapter) List(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, error) {
	return a.Service.ListBySuffix(ctx, suffix, filters, config)
}

// ListAndCount calls listAndCount<suffix>.
func (a MedusaServiceAdapter) ListAndCount(ctx context.Context, suffix string, filters *dal.Record, config utilsmodulessdk.FindConfig) ([]*dal.Record, int, error) {
	return a.Service.ListAndCountBySuffix(ctx, suffix, filters, config)
}
