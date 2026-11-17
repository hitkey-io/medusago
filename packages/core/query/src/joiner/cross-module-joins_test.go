package joiner

import (
	"encoding/json"
	"testing"

	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// Cases of cross-module-joins.spec.ts ("stage 1 pushdown") run over the
// runtime joiner configs of the default modules.

func mustRecord(t *testing.T, s string) *dal.Record {
	t.Helper()
	v, err := dal.DecodeOrderedJSON([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	r, ok := v.(*dal.Record)
	if !ok {
		t.Fatalf("not an object: %s", s)
	}
	return r
}

func specsJSON(t *testing.T, specs []*dal.CrossModuleJoinSpec) string {
	t.Helper()
	out := []any{}
	for _, s := range specs {
		spec := dal.NewRecord()
		if s.Parent != "" {
			spec.Set("parent", s.Parent)
		}
		link := dal.NewRecord()
		if s.Link.Schema != "" {
			link.Set("schema", s.Link.Schema)
		}
		link.Set("table", s.Link.Table)
		link.Set("sourceKey", s.Link.SourceKey)
		link.Set("targetKey", s.Link.TargetKey)
		spec.Set("link", link)
		target := dal.NewRecord()
		if s.Target.Schema != "" {
			target.Set("schema", s.Target.Schema)
		}
		target.Set("table", s.Target.Table)
		target.Set("primaryKey", s.Target.PrimaryKey)
		if s.Target.Filters != nil {
			target.Set("filters", s.Target.Filters)
		}
		spec.Set("target", target)
		out = append(out, spec)
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func compactJSON(t *testing.T, s string) string {
	t.Helper()
	v, err := dal.DecodeOrderedJSON([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRewriteNestedRelationFilters(t *testing.T) {
	catalog, err := NewGraphCatalog(modulessdk.DefaultJoinerConfigs, modulessdk.DefaultRelationMap)
	if err != nil {
		t.Fatal(err)
	}
	serviceConfig := catalog.GetServiceConfig("", "variant", "")
	if serviceConfig == nil {
		t.Fatal("variant service config not found")
	}

	variantPriceSet := `{"parent":"product_variant","link":{"table":"product_variant_price_set","sourceKey":"variant_id","targetKey":"price_set_id"},"target":{"table":"price_set","primaryKey":"id","filters":{"id":"pset_1"}}}`
	cases := []struct {
		name     string
		filters  string
		joins    string
		residual int
		left     string
	}{
		{
			name:    "pushes link filters nested below several module-internal relations",
			filters: `{"product":{"variants":{"price_set":{"id":"pset_1"}}}}`,
			joins: `[{"link":{"table":"product_variant","sourceKey":"id","targetKey":"product_id"},"target":{"table":"product","primaryKey":"id"}},` +
				`{"parent":"product","link":{"table":"product_variant","sourceKey":"product_id","targetKey":"id"},"target":{"table":"product_variant","primaryKey":"id"}},` +
				variantPriceSet + `]`,
			left: `{}`,
		},
		{
			name:    "keeps hasMany sibling filters on the same row as the nested link filter",
			filters: `{"product":{"title":"Shirt","variants":{"sku":"SHIRT-S","price_set":{"id":"pset_1"}}}}`,
			joins: `[{"link":{"table":"product_variant","sourceKey":"id","targetKey":"product_id"},"target":{"table":"product","primaryKey":"id"}},` +
				`{"parent":"product","link":{"table":"product_variant","sourceKey":"product_id","targetKey":"id"},"target":{"table":"product_variant","primaryKey":"id","filters":{"sku":"SHIRT-S"}}},` +
				variantPriceSet + `]`,
			left: `{"product":{"title":"Shirt"}}`,
		},
		{
			name:    "leaves hasMany filters in place when a sibling cannot be pushed down",
			filters: `{"product":{"variants":{"sku":{"$exists":true},"price_set":{"id":"pset_1"}}}}`,
			joins:   `[]`,
			left:    `{"product":{"variants":{"sku":{"$exists":true},"price_set":{"id":"pset_1"}}}}`,
		},
		{
			name:    "leaves nested link filters in place when they cannot be pushed down",
			filters: `{"product":{"variants":{"price_set":{"calculated_price":10}}}}`,
			joins:   `[]`,
			left:    `{"product":{"variants":{"price_set":{"calculated_price":10}}}}`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			filters := mustRecord(t, c.filters)
			query := &RemoteJoinerQuery{Alias: "variant", Fields: []string{"id"}, Args: []Arg{{Name: "filters", Value: filters}}}
			joins, residual, _ := extractCrossModuleJoins(query, serviceConfig, catalog)
			if got := specsJSON(t, joins); got != compactJSON(t, c.joins) {
				t.Errorf("crossModuleJoins\n got: %s\nwant: %s", got, compactJSON(t, c.joins))
			}
			if len(residual) != c.residual {
				t.Errorf("residual: got %d, want %d", len(residual), c.residual)
			}
			b, _ := json.Marshal(getFiltersArg(query.Args).Value)
			if string(b) != compactJSON(t, c.left) {
				t.Errorf("filters left\n got: %s\nwant: %s", b, compactJSON(t, c.left))
			}
		})
	}
}
