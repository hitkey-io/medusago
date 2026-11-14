package blog

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// This model only illustrates the indexes chapter: it is not part of the
// blog module.

// region: indexes
var Listing = dml.Define("listing", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "lst"}).PrimaryKey()},
	{Name: "title", Property: dml.Text()},
	{Name: "status", Property: dml.Enum("draft", "published").Default("draft")},
	{Name: "category", Property: dml.Text()},
	{Name: "published_at", Property: dml.DateTime().Nullable()},
	{Name: "views", Property: dml.Number().Default(0)},
}).WithIndexes([]dml.Index{
	// one index over two properties
	{On: []string{"category", "status"}},
	// with a name
	{Name: "IDX_listing_recent", On: []string{"published_at"}},
	// a unique index
	{On: []string{"title", "category"}, Unique: true},
	// a partial index
	{On: []string{"views"}, Where: "status = 'published'"},
	// an index of a type other than btree
	{On: []string{"published_at"}, Type: "brin"},
	// an index that also covers soft-deleted records
	{Name: "IDX_listing_status_all", On: []string{"status"}, WhereNull: true},
})

// endregion: indexes

// region: checks
var Discount = dml.Define("discount", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "dsc"}).PrimaryKey()},
	{Name: "percentage", Property: dml.Number()},
	{Name: "starts_at", Property: dml.DateTime()},
	{Name: "ends_at", Property: dml.DateTime()},
}).WithChecks([]dml.Check{
	{Name: "discount_percentage_range", Expression: "percentage between 1 and 100"},
	{Expression: "ends_at > starts_at"},
})

// endregion: checks
