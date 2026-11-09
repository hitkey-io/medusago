package utils

import (
	"context"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// QueryOf resolves Query from a container.
func QueryOf(c medusahttp.MedusaContainer) *query.Query {
	q, _ := c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	return q
}

// RefetchEntitiesInput mirrors the input of refetchEntities.
type RefetchEntitiesInput struct {
	Entity string
	// IDOrFilter is a string id or a filters record.
	IDOrFilter  any
	Fields      []string
	Pagination  *dal.Record
	WithDeleted bool
}

// RefetchEntities mirrors refetchEntities (the context filter is not
// supported yet).
func RefetchEntities(ctx context.Context, scope medusahttp.MedusaContainer, in RefetchEntitiesInput) (*query.GraphResult, error) {
	var filters *dal.Record
	switch t := in.IDOrFilter.(type) {
	case string:
		filters = dal.NewRecord()
		filters.Set("id", t)
	case *dal.Record:
		filters = dal.NewRecord()
		for _, k := range t.Keys() {
			if k == "context" {
				continue
			}
			v, _ := t.Get(k)
			filters.Set(k, v)
		}
	}
	fields := in.Fields
	if fields == nil {
		fields = []string{}
	}
	res, err := QueryOf(scope).Graph(ctx, query.GraphInput{Entity: in.Entity, Fields: fields, Filters: filters, Pagination: in.Pagination, WithDeleted: in.WithDeleted}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	if res.Metadata == nil {
		res.Metadata = dal.NewRecord()
	}
	return res, nil
}

// RefetchEntity mirrors refetchEntity: the first row or nil.
func RefetchEntity(ctx context.Context, scope medusahttp.MedusaContainer, entity string, idOrFilter any, fields []string) (any, error) {
	res, err := RefetchEntities(ctx, scope, RefetchEntitiesInput{Entity: entity, IDOrFilter: idOrFilter, Fields: fields})
	if err != nil {
		return nil, err
	}
	if len(res.Data) == 0 {
		return nil, nil
	}
	return res.Data[0], nil
}

// ListVariables builds the { filters, ...pagination } variables of the
// list routes.
func ListVariables(filters *dal.Record, pagination *dal.Record) *dal.Record {
	v := dal.NewRecord()
	if filters == nil {
		filters = dal.NewRecord()
	}
	v.Set("filters", filters)
	if pagination != nil {
		for _, k := range pagination.Keys() {
			x, _ := pagination.Get(k)
			v.Set(k, x)
		}
	}
	return v
}

// RemoteQueryList mirrors remoteQuery(remoteQueryObjectFromString({
// entryPoint, variables, fields })).
func RemoteQueryList(ctx context.Context, scope medusahttp.MedusaContainer, entryPoint string, variables *dal.Record, fields []string) (*query.GraphResult, error) {
	res, err := QueryOf(scope).RemoteQuery(ctx, query.RemoteQueryObjectFromString(query.RemoteQueryObjectConfig{EntryPoint: entryPoint, Variables: variables, Fields: fields}))
	if err != nil {
		return nil, err
	}
	if res.Metadata == nil {
		res.Metadata = dal.NewRecord()
	}
	return res, nil
}

// MetaValue returns metadata[key] (undefined -> nil).
func MetaValue(meta *dal.Record, key string) any {
	if meta == nil {
		return nil
	}
	v, _ := meta.Get(key)
	return v
}

// ListResponse builds { <key>: rows, count, offset, limit }.
func ListResponse(key string, res *query.GraphResult) *dal.Record {
	out := dal.NewRecord()
	out.Set(key, res.Data)
	if v, ok := res.Metadata.Get("count"); ok {
		out.Set("count", v)
	}
	if v, ok := res.Metadata.Get("skip"); ok {
		out.Set("offset", v)
	}
	if v, ok := res.Metadata.Get("take"); ok {
		out.Set("limit", v)
	}
	return out
}
