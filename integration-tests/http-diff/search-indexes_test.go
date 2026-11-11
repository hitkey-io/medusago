package httpdiff

import (
	"context"
	"os"

	medusahttp "github.com/hitkey-io/medusago/packages/core/framework/src/http"
	"github.com/hitkey-io/medusago/packages/core/framework/src/http/middlewares"
	coremodulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	searchtypes "github.com/hitkey-io/medusago/packages/core/types/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	searchutils "github.com/hitkey-io/medusago/packages/core/utils/src/search"
	"github.com/hitkey-io/medusago/packages/core/utils/src/search/model"
)

// The test search indexes of the reference application
// (src/search/product.js and src/api/middlewares.js of its copy), declared
// when the scenario environment sets MEDUSA_TEST_SEARCH=true.

func recordList(v any) []*dal.Record {
	var out []*dal.Record
	switch t := v.(type) {
	case []any:
		for _, e := range t {
			if r, ok := e.(*dal.Record); ok {
				out = append(out, r)
			}
		}
	case []*dal.Record:
		out = t
	}
	return out
}

func field(r *dal.Record, key string) any {
	if r == nil {
		return nil
	}
	v, _ := r.Get(key)
	return v
}

func toProductDocuments(_ context.Context, rows []*dal.Record, _ searchtypes.IngestionContext) ([]*dal.Record, error) {
	out := make([]*dal.Record, 0, len(rows))
	for _, row := range rows {
		d := dal.NewRecord()
		for _, k := range []string{"id", "title", "subtitle", "description", "handle", "status", "is_giftcard", "discountable", "created_at"} {
			d.Set(k, field(row, k))
		}
		channels := []any{}
		for _, sc := range recordList(field(row, "sales_channels")) {
			channels = append(channels, field(sc, "id"))
		}
		d.Set("sales_channel_ids", channels)
		var collectionTitle any
		if c, ok := field(row, "collection").(*dal.Record); ok {
			collectionTitle = field(c, "title")
		}
		d.Set("collection_title", collectionTitle)
		variants := recordList(field(row, "variants"))
		d.Set("variant_count", float64(len(variants)))
		tags := []any{}
		for _, tag := range recordList(field(row, "tags")) {
			t := dal.NewRecord()
			t.Set("value", field(tag, "value"))
			tags = append(tags, t)
		}
		d.Set("tags", tags)
		vs := []any{}
		for _, variant := range variants {
			v := dal.NewRecord()
			v.Set("sku", field(variant, "sku"))
			v.Set("title", field(variant, "title"))
			vs = append(vs, v)
		}
		d.Set("variants", vs)
		out = append(out, d)
	}
	return out, nil
}

var productGraphFields = []string{
	"id", "title", "subtitle", "description", "handle", "status", "is_giftcard", "discountable", "created_at",
	"tags.value", "variants.sku", "variants.title", "sales_channels.id", "collection.title",
}

// registerTestSearchIndexes resets the search index registry and, with
// MEDUSA_TEST_SEARCH=true, declares the test indexes. It returns the
// project middlewares to load.
func registerTestSearchIndexes() []medusahttp.MiddlewareRoute {
	coremodulessdk.ClearSearchIndexes()
	if os.Getenv("MEDUSA_TEST_SEARCH") != "true" {
		return nil
	}
	source := searchutils.GraphSourceOptions{Fields: productGraphFields, Transform: toProductDocuments}
	searchutils.DefineSearchIndex(searchtypes.Definition{
		Name:   "product",
		Entity: "product",
		Fields: model.Define(
			model.Field("id", model.Keyword().Filterable()),
			model.Field("title", model.Text().SearchableWeight(3)),
			model.Field("subtitle", model.Text().Searchable()),
			model.Field("description", model.Text().SearchableWeight(1).Retrievable(false)),
			model.Field("handle", model.Keyword().Filterable().Sortable()),
			model.Field("status", model.Keyword().Filterable().Facetable()),
			model.Field("is_giftcard", model.Boolean().Filterable().Facetable()),
			model.Field("discountable", model.Boolean().Filterable().Sortable()),
			model.Field("created_at", model.Date().Filterable().Sortable().Facetable()),
			model.Field("sales_channel_ids", model.Keyword().Array().Filterable()),
			model.Field("collection_title", model.Keyword().Filterable().Facetable()),
			model.Field("variant_count", model.Integer().Filterable().Sortable().FacetableTypes("range", "stats")),
			model.Field("tags", model.Object(model.Field("value", model.Keyword().Filterable().Facetable())).Array()),
			model.Field("variants", model.Object(
				model.Field("sku", model.Keyword().Searchable().Filterable()),
				model.Field("title", model.Text().SearchableWeight(2)),
			).Array()),
		),
		Events:  []string{"product.created", "product.updated", "product.deleted"},
		Consume: searchutils.GraphConsume(searchutils.GraphConsumeOptions{GraphSourceOptions: source}),
		Seed:    searchutils.GraphSeed(searchutils.GraphSeedOptions{GraphSourceOptions: source}),
	})
	searchutils.DefineSearchIndex(searchtypes.Definition{
		Name:   "collection",
		Entity: "product_collection",
		Fields: model.Define(
			model.Field("id", model.Keyword().Filterable()),
			model.Field("title", model.Text().Searchable()),
			model.Field("handle", model.Keyword().Filterable().Sortable()),
		),
		Seed: searchutils.GraphSeed(searchutils.GraphSeedOptions{GraphSourceOptions: searchutils.GraphSourceOptions{Fields: []string{"id", "title", "handle"}}}),
	})
	return []medusahttp.MiddlewareRoute{{
		Matcher: "/store/search",
		Middlewares: []medusahttp.MiddlewareFunction{
			middlewares.ConfigureStoreSearch(middlewares.StoreSearchConfig{AllowedIndexes: map[string]any{"product": true}}),
		},
	}}
}
