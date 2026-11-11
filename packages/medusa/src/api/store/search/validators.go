// Package search mirrors api/store/search.
package search

import (
	"strings"

	"github.com/hitkey-io/medusago/packages/core/framework/src/zod"
)

// maxSearchQueries mirrors MAX_SEARCH_QUERIES.
const maxSearchQueries = 20

// maxSearchTake mirrors MAX_SEARCH_TAKE.
const maxSearchTake = 100

func filterValue() *zod.Schema {
	return zod.Union(zod.String(), zod.Number(), zod.Boolean(), zod.Null())
}

var storeSearchOperatorMap = zod.Object(
	zod.F("$eq", filterValue().Optional()),
	zod.F("$ne", filterValue().Optional()),
	zod.F("$in", zod.Array(filterValue()).Optional()),
	zod.F("$nin", zod.Array(filterValue()).Optional()),
	zod.F("$lt", filterValue().Optional()),
	zod.F("$lte", filterValue().Optional()),
	zod.F("$gt", filterValue().Optional()),
	zod.F("$gte", filterValue().Optional()),
	zod.F("$exists", zod.Boolean().Optional()),
	zod.F("$contains", zod.Union(filterValue(), zod.Array(filterValue())).Optional()),
	zod.F("$overlaps", zod.Array(filterValue()).Optional()),
	zod.F("$prefix", zod.String().Optional()),
	zod.F("$like", zod.String().Optional()),
).Strict()

// storeSearchFilters mirrors StoreSearchFilters: a filter tree keyed by
// the index' own fields (recursive, built on first use).
var storeSearchFilters *zod.Schema

func filtersSchema() *zod.Schema {
	if storeSearchFilters == nil {
		storeSearchFilters = zod.Lazy(func() *zod.Schema {
			return zod.Record(zod.String(), zod.Union(
				filterValue(),
				zod.Array(filterValue()),
				storeSearchOperatorMap,
				storeSearchFilters,
				zod.Array(storeSearchFilters),
			))
		})
	}
	return storeSearchFilters
}

var storeSearchFacet = zod.Union(
	zod.String(),
	zod.Object(
		zod.F("field", zod.String()),
		zod.F("type", zod.Literal("value").Optional()),
		zod.F("limit", zod.Number().Int().Gt(0).Optional()),
		zod.F("sort", zod.Enum("count", "alpha").Optional()),
		zod.F("query", zod.String().Optional()),
	).Strict(),
	zod.Object(
		zod.F("field", zod.String()),
		zod.F("type", zod.Literal("range")),
		zod.F("ranges", zod.Array(zod.Object(
			zod.F("key", zod.String().Optional()),
			zod.F("from", zod.Union(zod.Number(), zod.String()).Optional()),
			zod.F("to", zod.Union(zod.Number(), zod.String()).Optional()),
		).Strict())),
	).Strict(),
	zod.Object(
		zod.F("field", zod.String()),
		zod.F("type", zod.Literal("stats")),
	).Strict(),
)

var storeSearchHighlight = zod.Union(
	zod.Boolean(),
	zod.Object(
		zod.F("fields", zod.Array(zod.String())),
		zod.F("pre_tag", zod.String().Optional()),
		zod.F("post_tag", zod.String().Optional()),
		zod.F("snippet", zod.Union(zod.Boolean(), zod.Object(zod.F("length", zod.Number().Int().Gt(0)))).Optional()),
	).Strict(),
)

// No provider_options: it reaches straight into the engine's own API.
var storeSearchOptions = zod.Object(
	zod.F("attributes_to_search_on", zod.Array(zod.String()).Optional()),
	zod.F("match_strategy", zod.Enum("all", "any", "last").Optional()),
	zod.F("typo_tolerance", zod.Boolean().Optional()),
	zod.F("facets", zod.Array(storeSearchFacet).Optional()),
	zod.F("disjunctive_facets", zod.Boolean().Optional()),
	zod.F("highlight", storeSearchHighlight.Optional()),
	zod.F("distinct", zod.String().Optional()),
	zod.F("min_score", zod.Number().Optional()),
	zod.F("include_score", zod.Boolean().Optional()),
	zod.F("locales", zod.Array(zod.String()).Optional()),
	zod.F("vector", zod.Object(
		zod.F("field", zod.String().Optional()),
		zod.F("value", zod.Array(zod.Number()).Optional()),
		zod.F("query", zod.String().Optional()),
		zod.F("semantic_ratio", zod.Number().Min(0).Max(1).Optional()),
	).Strict().Optional()),
	zod.F("count", zod.Enum("estimated", "exact", "none").Optional()),
).Strict()

func toUpperCase(v any) any {
	if s, ok := v.(string); ok {
		return strings.ToUpper(s)
	}
	return v
}

var storeSearchPagination = zod.Object(
	zod.F("skip", zod.Number().Int().Nonnegative().Optional()),
	zod.F("take", zod.Number().Int().Nonnegative().Max(maxSearchTake).Optional()),
	zod.F("order", zod.Record(zod.String(), zod.Preprocess(toUpperCase, zod.Enum("ASC", "DESC"))).Optional()),
	zod.F("cursor", zod.String().Optional()),
).Strict()

func storeSearchQuery() *zod.Schema {
	return zod.Object(
		zod.F("entity", zod.String().Min(1)),
		zod.F("fields", zod.Array(zod.String()).Optional()),
		zod.F("filters", filtersSchema().Optional()),
		zod.F("pagination", storeSearchPagination.Optional()),
		zod.F("search_options", storeSearchOptions.Optional()),
	).Strict()
}

// StoreSearch mirrors StoreSearch: a batch of queries, or a single one.
var StoreSearch = zod.Union(
	zod.Object(
		zod.F("queries", zod.Array(storeSearchQuery()).Min(1).Max(maxSearchQueries)),
	).Strict(),
	storeSearchQuery(),
)
