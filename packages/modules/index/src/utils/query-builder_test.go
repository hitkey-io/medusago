package utils

import (
	"os"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// TestQueryBuilderVectors compares the SQL built for query.index() inputs
// with the SQL upstream's QueryBuilder (knex, pg) built for the same
// inputs (testdata/query-builder-vectors.json, generated in the reference
// application over the default schema).
func TestQueryBuilderVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/query-builder-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := dal.DecodeOrderedJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range decoded.([]any) {
		vec := v.(*dal.Record)
		name := get(vec, "name").(string)
		t.Run(name, func(t *testing.T) {
			schema, err := BuildSchemaObjectRepresentation()
			if err != nil {
				t.Fatal(err)
			}
			input := get(vec, "input").(*dal.Record)
			entity := get(input, "entity").(string)
			// Query.index
			filters := dal.NewRecord()
			if f, ok := get(input, "filters").(*dal.Record); ok {
				filters.Set(entity, f)
			}
			joinFilters := dal.NewRecord()
			if f, ok := get(input, "joinFilters").(*dal.Record); ok {
				joinFilters.Set(entity, f)
			}
			pagination, _ := get(input, "pagination").(*dal.Record)
			if pagination != nil {
				if o, ok := get(pagination, "order").(*dal.Record); ok {
					wrapped := dal.NewRecord()
					wrapped.Set(entity, UnflattenObjectKeys(o))
					pagination.Set("order", wrapped)
				}
			}
			prepared := PrepareIndexQuery(schema, IndexQueryConfig{
				Fields:      []string{entity + ".id"},
				Filters:     filters,
				JoinFilters: joinFilters,
				Pagination:  pagination,
				IdsOnly:     true,
			})
			sql, sqlCount, err := prepared.Builder.BuildQuery(prepared.HasPagination, prepared.HasCount)
			if wantErr, ok := get(vec, "error").(string); ok {
				if err == nil || err.Error() != wantErr {
					t.Fatalf("error: want %q, got %v", wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if want := get(vec, "sql").(string); sql != want {
				t.Errorf("sql\nwant %s\ngot  %s", want, sql)
			}
			wantCount, _ := get(vec, "sqlCount").(string)
			if sqlCount != wantCount {
				t.Errorf("sqlCount\nwant %s\ngot  %s", wantCount, sqlCount)
			}
		})
	}
}
