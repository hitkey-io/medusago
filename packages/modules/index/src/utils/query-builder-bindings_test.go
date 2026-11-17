package utils

import (
	"strings"
	"testing"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// TestQueryBuilderQuestionMarks mirrors the "question marks in filter
// values" cases of query-builder.spec.ts (v2.22.7): the join parts and the
// paginated inner query keep their values bound, so a `?` inside a value
// is not taken for a placeholder when the outer query is formatted.
func TestQueryBuilderQuestionMarks(t *testing.T) {
	schema, err := loadSchemaObjectRepresentation("testdata/index-schemas/default.json")
	if err != nil {
		t.Fatal(err)
	}
	build := func(filters, joinFilters string) string {
		t.Helper()
		f := dal.NewRecord()
		if filters != "" {
			v, err := dal.DecodeOrderedJSON([]byte(filters))
			if err != nil {
				t.Fatal(err)
			}
			f.Set("product", v)
		}
		jf := dal.NewRecord()
		if joinFilters != "" {
			v, err := dal.DecodeOrderedJSON([]byte(joinFilters))
			if err != nil {
				t.Fatal(err)
			}
			jf.Set("product", v)
		}
		pagination := dal.NewRecord()
		pagination.Set("skip", float64(0))
		pagination.Set("take", float64(20))
		prepared := PrepareIndexQuery(schema, IndexQueryConfig{
			Fields:      []string{"product.id", "product.title", "product.variants.id"},
			Filters:     f,
			JoinFilters: jf,
			Pagination:  pagination,
		})
		sql, _, err := prepared.Builder.BuildQuery(prepared.HasPagination, prepared.HasCount)
		if err != nil {
			t.Fatal(err)
		}
		return sql
	}

	sql := build(`{"title":"What? Product","id":{"$ne":"it's a product"}}`, "")
	if n := strings.Count(sql, `What? Product`); n != 2 {
		t.Errorf("title value: want 2 occurrences, got %d in\n%s", n, sql)
	}
	if n := strings.Count(sql, `'it''s a product'`); n != 2 {
		t.Errorf("id value: want 2 occurrences, got %d in\n%s", n, sql)
	}
	if strings.Contains(sql, "What'it") || strings.Contains(sql, "What"+"'") {
		t.Errorf("a value was shifted into the question mark:\n%s", sql)
	}

	sql = build(`{"id":"prod_1"}`, `{"variants.sku":"sku?"}`)
	if !strings.Contains(sql, `sku?`) {
		t.Errorf("join filter value lost:\n%s", sql)
	}
	if n := strings.Count(sql, `'prod_1'`); n != 2 {
		t.Errorf("id value: want 2 occurrences, got %d in\n%s", n, sql)
	}
}
