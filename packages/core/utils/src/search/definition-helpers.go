package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// The timestamps every Medusa entity carries, which the catch-up pass reads.
const (
	updatedAtField = "updated_at"
	deletedAtField = "deleted_at"
)

// GraphSourceOptions mirrors SearchGraphSourceOptions.
type GraphSourceOptions struct {
	// Entity is the query.graph entry point ("" = the index entity).
	Entity string
	// Fields are the fields selected from query.graph.
	Fields []string
	// Context returns the query context the rows are read with (nil: none).
	Context func(ic searchtypes.IngestionContext) *dal.Record
	// Transform maps a page of rows to the documents to index (nil: the
	// rows as-is). Every document must carry the row's key as `id`.
	Transform func(ctx context.Context, rows []*dal.Record, ic searchtypes.IngestionContext) ([]*dal.Record, error)
}

// GraphSeedOptions mirrors SearchGraphSeedOptions.
type GraphSeedOptions struct {
	GraphSourceOptions
	// BatchSize is the number of rows per page (0 = 200).
	BatchSize int
}

// GraphConsumeOptions mirrors SearchGraphConsumeOptions.
type GraphConsumeOptions struct {
	GraphSourceOptions
	// ResolveIDs returns the ids the event affects (nil: event.data.id).
	ResolveIDs func(ctx context.Context, event searchtypes.Event, ic searchtypes.IngestionContext) (any, error)
	// IsDelete reports whether the event removes its documents (nil: the
	// event name ends in ".deleted").
	IsDelete func(event searchtypes.Event) bool
}

func toArray(value any) []string {
	if value == nil {
		return nil
	}
	values, ok := asArray(value)
	if !ok {
		values = []any{value}
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = JSString(v)
	}
	return out
}

// seedFilters mirrors seedFilters: the reindex filters narrowed by the
// catch-up window and the paging cursor (under $and).
func seedFilters(filters *dal.Record, constraints []*dal.Record) *dal.Record {
	if len(constraints) == 0 {
		if filters == nil {
			return dal.NewRecord()
		}
		return filters
	}
	if filters == nil || filters.Len() == 0 {
		out := dal.NewRecord()
		for _, c := range constraints {
			for _, k := range c.Keys() {
				v, _ := c.Get(k)
				out.Set(k, v)
			}
		}
		return out
	}
	var existing []any
	if v, ok := filters.Get("$and"); ok {
		existing, _ = v.([]any)
	}
	and := append([]any{}, existing...)
	for _, c := range constraints {
		and = append(and, c)
	}
	out := dal.NewRecord()
	for _, k := range filters.Keys() {
		v, _ := filters.Get(k)
		out.Set(k, v)
	}
	out.Set("$and", and)
	return out
}

func withFields(fields []string, extra ...string) []string {
	out := fields
	for _, f := range extra {
		found := false
		for _, e := range out {
			if e == f {
				found = true
				break
			}
		}
		if !found {
			out = append(append([]string{}, out...), f)
		}
	}
	return out
}

// toDocuments mirrors toDocuments: the documents of a page of rows and the
// keys they were produced for.
func toDocuments(ctx context.Context, rows []*dal.Record, options GraphSourceOptions, ic searchtypes.IngestionContext) ([]*dal.Record, map[string]bool, error) {
	produced := map[string]bool{}
	if options.Transform == nil {
		for _, row := range rows {
			v, _ := row.Get(ic.Index.PrimaryKey)
			produced[JSString(v)] = true
		}
		return rows, produced, nil
	}
	documents, err := options.Transform(ctx, rows, ic)
	if err != nil {
		return nil, nil, err
	}
	for _, document := range documents {
		id, ok := document.Get("id")
		if !ok || id == nil {
			return nil, nil, fmt.Errorf(`Search index "%s": transform returned a document without an "id"`, ic.Index.Name)
		}
		produced[JSString(id)] = true
	}
	return documents, produced, nil
}

func graphQuery(c searchtypes.Container) (*query.Query, error) {
	q, ok := c.Query.(*query.Query)
	if !ok || q == nil {
		return nil, fmt.Errorf("Cannot read properties of undefined (reading 'graph')")
	}
	return q, nil
}

func graphRows(data []any) []*dal.Record {
	out := make([]*dal.Record, 0, len(data))
	for _, d := range data {
		if r, ok := d.(*dal.Record); ok {
			out = append(out, r)
		}
	}
	return out
}

func entityOf(options GraphSourceOptions, index *searchtypes.ResolvedDefinition) string {
	if options.Entity != "" {
		return options.Entity
	}
	return index.Entity
}

func queryContext(options GraphSourceOptions, ic searchtypes.IngestionContext) *dal.Record {
	if options.Context == nil {
		return nil
	}
	return options.Context(ic)
}

// GraphSeed mirrors graphSeed: a `seed` paging the entity through
// query.graph in ascending primary key order (resuming after last_key);
// the catch-up pass reads the rows touched since it started, soft-deleted
// ones included, and turns the ones without a document into deletes.
func GraphSeed(options GraphSeedOptions) searchtypes.SeedFunc {
	batchSize := options.BatchSize
	if batchSize == 0 {
		batchSize = 200
	}
	return func(ctx context.Context, sc searchtypes.SeedContext, yield func([]searchtypes.Mutation) error) error {
		q, err := graphQuery(sc.Container)
		if err != nil {
			return err
		}
		index := sc.Index
		entity := entityOf(options.GraphSourceOptions, index)
		primaryKey := index.PrimaryKey
		selection := withFields(options.Fields, primaryKey)
		if sc.Catchup != nil {
			selection = withFields(options.Fields, primaryKey, deletedAtField)
		}
		var cursor *string
		if sc.LastKey != nil {
			k := *sc.LastKey
			cursor = &k
		}
		for {
			var constraints []*dal.Record
			if sc.Catchup != nil {
				gte := dal.NewRecord()
				gte.Set("$gte", sc.Catchup.Since)
				c := dal.NewRecord()
				c.Set(updatedAtField, gte)
				constraints = append(constraints, c)
			}
			if cursor != nil {
				gt := dal.NewRecord()
				gt.Set("$gt", *cursor)
				c := dal.NewRecord()
				c.Set(primaryKey, gt)
				constraints = append(constraints, c)
			}
			order := dal.NewRecord()
			order.Set(primaryKey, "ASC")
			pagination := dal.NewRecord()
			pagination.Set("take", float64(batchSize))
			pagination.Set("order", order)
			res, err := q.Graph(ctx, query.GraphInput{
				Entity:      entity,
				Fields:      selection,
				Filters:     seedFilters(sc.Filters, constraints),
				Pagination:  pagination,
				WithDeleted: sc.Catchup != nil,
				Context:     queryContext(options.GraphSourceOptions, sc.IngestionContext),
			}, joiner.Options{})
			if err != nil {
				return err
			}
			data := graphRows(res.Data)
			if len(data) == 0 {
				return nil
			}
			rows := data
			if sc.Catchup != nil {
				rows = nil
				for _, row := range data {
					if v, _ := row.Get(deletedAtField); !Truthy(v) {
						rows = append(rows, row)
					}
				}
			}
			documents, produced, err := toDocuments(ctx, rows, options.GraphSourceOptions, sc.IngestionContext)
			if err != nil {
				return err
			}
			var deletedIDs []any
			if sc.Catchup != nil {
				for _, row := range data {
					v, _ := row.Get(primaryKey)
					id := JSString(v)
					if !produced[id] {
						deletedIDs = append(deletedIDs, id)
					}
				}
			}
			var mutations []searchtypes.Mutation
			if len(documents) > 0 {
				mutations = append(mutations, searchtypes.Mutation{Action: "upsert", Documents: documents})
			}
			if len(deletedIDs) > 0 {
				f := dal.NewRecord()
				f.Set(primaryKey, deletedIDs)
				mutations = append(mutations, searchtypes.Mutation{Action: "delete", Filters: f})
			}
			if len(mutations) > 0 {
				if err := yield(mutations); err != nil {
					return err
				}
			}
			if len(data) < batchSize {
				return nil
			}
			last, _ := data[len(data)-1].Get(primaryKey)
			k := JSString(last)
			cursor = &k
		}
	}
}

// GraphConsume mirrors graphConsume: the event's ids are read back through
// query.graph and upserted; an id the read does not return (or the
// transform rejects) is deleted instead.
func GraphConsume(options GraphConsumeOptions) searchtypes.ConsumeFunc {
	return func(ctx context.Context, event searchtypes.Event, ic searchtypes.IngestionContext) ([]searchtypes.Mutation, error) {
		index := ic.Index
		entity := entityOf(options.GraphSourceOptions, index)
		primaryKey := index.PrimaryKey
		var raw any
		if options.ResolveIDs != nil {
			v, err := options.ResolveIDs(ctx, event, ic)
			if err != nil {
				return nil, err
			}
			raw = v
		} else {
			raw, _ = get(event.Data, "id")
		}
		ids := toArray(raw)
		if len(ids) == 0 {
			return []searchtypes.Mutation{}, nil
		}
		isDelete := strings.HasSuffix(event.Name, ".deleted")
		if options.IsDelete != nil {
			isDelete = options.IsDelete(event)
		}
		idValues := make([]any, len(ids))
		for i, id := range ids {
			idValues[i] = id
		}
		if isDelete {
			f := dal.NewRecord()
			f.Set(primaryKey, idValues)
			return []searchtypes.Mutation{{Action: "delete", Filters: f}}, nil
		}
		q, err := graphQuery(ic.Container)
		if err != nil {
			return nil, err
		}
		filters := dal.NewRecord()
		filters.Set(primaryKey, idValues)
		res, err := q.Graph(ctx, query.GraphInput{
			Entity:  entity,
			Fields:  withFields(options.Fields, primaryKey),
			Filters: filters,
			Context: queryContext(options.GraphSourceOptions, ic),
		}, joiner.Options{})
		if err != nil {
			return nil, err
		}
		documents, produced, err := toDocuments(ctx, graphRows(res.Data), options.GraphSourceOptions, ic)
		if err != nil {
			return nil, err
		}
		var mutations []searchtypes.Mutation
		if len(documents) > 0 {
			mutations = append(mutations, searchtypes.Mutation{Action: "upsert", Documents: documents})
		}
		var removed []any
		for _, id := range ids {
			if !produced[id] {
				removed = append(removed, id)
			}
		}
		if len(removed) > 0 {
			f := dal.NewRecord()
			f.Set(primaryKey, removed)
			mutations = append(mutations, searchtypes.Mutation{Action: "delete", Filters: f})
		}
		return mutations, nil
	}
}
