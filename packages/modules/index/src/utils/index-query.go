package utils

import (
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// IndexQueryConfig mirrors IndexTypes.IndexQueryConfig.
type IndexQueryConfig struct {
	Fields      []string
	Filters     *dal.Record
	JoinFilters *dal.Record
	// Pagination holds skip, take and order (keyed by the entity).
	Pagination *dal.Record
	IdsOnly    bool
}

// PreparedIndexQuery is the query builder PostgresProvider.query builds
// for a config, with its pagination state.
type PreparedIndexQuery struct {
	Builder       *QueryBuilder
	HasPagination bool
	HasCount      bool
	Skip          any
	Take          any
}

// PrepareIndexQuery mirrors the first half of PostgresProvider.query: the
// selection, filters and ordering normalisation and the QueryBuilder.
func PrepareIndexQuery(schema *SchemaObjectRepresentation, config IndexQueryConfig) *PreparedIndexQuery {
	filters := config.Filters
	if filters == nil {
		filters = dal.NewRecord()
	}
	joinFilters := config.JoinFilters
	if joinFilters == nil {
		joinFilters = dal.NewRecord()
	}
	var take, skip any
	inputOrderBy := dal.NewRecord()
	hasTake, hasSkip := false, false
	if p := config.Pagination; p != nil {
		take, hasTake = p.Get("take")
		skip, hasSkip = p.Get("skip")
		hasTake = hasTake && take != nil
		hasSkip = hasSkip && skip != nil
		if o, ok := p.Get("order"); ok && o != nil {
			if r, ok := o.(*dal.Record); ok {
				inputOrderBy = r
			}
		}
	}

	selection := NormalizeFieldsSelection(config.Fields)
	where := FlattenObjectKeys(UnflattenObjectKeys(filters))
	inputOrderByObj := UnflattenObjectKeys(inputOrderBy)
	joinWhere := FlattenObjectKeys(UnflattenObjectKeys(joinFilters))
	orderBy := FlattenObjectKeys(inputOrderByObj)

	out := &PreparedIndexQuery{Skip: skip, Take: take}
	if hasSkip || hasTake {
		out.HasPagination = true
		if hasSkip {
			out.HasCount = true
		}
	}
	requested, _ := DeepMerge(DeepMerge(selection, filters), inputOrderByObj).(*dal.Record)
	out.Builder = NewQueryBuilder(schema, QueryFormat{Select: selection, Where: where, JoinWhere: joinWhere}, QueryOptions{Skip: skip, Take: take, OrderBy: orderBy}, filters, requested, config.IdsOnly)
	return out
}
