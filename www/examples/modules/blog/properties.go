package blog

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// These models only illustrate the properties chapter: they are not part of
// the blog module.

// region: property-types
var Showcase = dml.Define("showcase", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "show"}).PrimaryKey()},
	{Name: "title", Property: dml.Text()},
	{Name: "views", Property: dml.Number()},
	{Name: "rating", Property: dml.Float()},
	{Name: "price", Property: dml.BigNumber()},
	{Name: "is_featured", Property: dml.Boolean()},
	{Name: "status", Property: dml.Enum("draft", "published")},
	{Name: "published_at", Property: dml.DateTime()},
	{Name: "settings", Property: dml.JSON()},
	{Name: "labels", Property: dml.Array()},
	{Name: "position", Property: dml.Autoincrement()},
})

// endregion: property-types

// region: property-modifiers
var Article = dml.Define("article", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "art"}).PrimaryKey()},
	// optional
	{Name: "subtitle", Property: dml.Text().Nullable()},
	// with a default value
	{Name: "views", Property: dml.Number().Default(0)},
	{Name: "status", Property: dml.Enum("draft", "published").Default("draft")},
	{Name: "labels", Property: dml.Array().Default([]any{"new"})},
	// unique, among the records that are not soft-deleted
	{Name: "slug", Property: dml.Text().Unique()},
	// with a database index
	{Name: "author_id", Property: dml.Text().Index().Nullable()},
	// the filter `q` searches the searchable properties
	{Name: "title", Property: dml.Text().Searchable()},
	// not stored in the database
	{Name: "calculated_price", Property: dml.BigNumber().Computed()},
})

// endregion: property-modifiers

// region: primary-key
var Product = dml.Define("my_product", dml.Schema{
	{Name: "sku", Property: dml.Text().PrimaryKey()},
	{Name: "name", Property: dml.Text()},
})

// endregion: primary-key

// region: composite-primary-key
var PostStat = dml.Define("post_stat", dml.Schema{
	{Name: "post_id", Property: dml.Text().PrimaryKey()},
	{Name: "period", Property: dml.Text().PrimaryKey()},
	{Name: "views", Property: dml.Number().Default(0)},
})

// endregion: composite-primary-key

// region: named-index
var Reference = dml.Define("reference", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "ref"}).PrimaryKey()},
	{Name: "code", Property: dml.Text().Unique("IDX_reference_code")},
	{Name: "owner_id", Property: dml.Text().Index("IDX_reference_owner")},
})

// endregion: named-index

// region: translatable
var Page = dml.Define("page", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "page"}).PrimaryKey()},
	{Name: "title", Property: dml.Text().Translatable()},
	{Name: "body", Property: dml.Text().Translatable()},
})

// endregion: translatable
