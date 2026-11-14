package blog

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// These models only illustrate the relationships chapter: they are not part
// of the blog module.

// region: one-to-one
var Author = dml.Define("author", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "author"}).PrimaryKey()},
	{Name: "name", Property: dml.Text()},
	{Name: "profile", Property: dml.HasOne("AuthorProfile", dml.RelationshipOptions{MappedBy: "author"})},
})

var AuthorProfile = dml.Define("author_profile", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "aprof"}).PrimaryKey()},
	{Name: "bio", Property: dml.Text().Nullable()},
	{Name: "author", Property: dml.BelongsTo("Author", dml.RelationshipOptions{MappedBy: "profile"})},
})

// endregion: one-to-one

// region: one-to-one-optional
var Writer = dml.Define("writer", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "writer"}).PrimaryKey()},
	{Name: "profile", Property: dml.HasOne("WriterProfile", dml.RelationshipOptions{MappedBy: "writer"}).Nullable()},
})

var WriterProfile = dml.Define("writer_profile", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "wprof"}).PrimaryKey()},
	// a profile may belong to no writer
	{Name: "writer", Property: dml.BelongsTo("Writer", dml.RelationshipOptions{MappedBy: "profile"}).Nullable()},
})

// endregion: one-to-one-optional

// region: one-sided-has-one
// The foreign key column is in the user table, and Email has no property
// pointing back to User.
var User = dml.Define("user_account", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "usr"}).PrimaryKey()},
	{Name: "email", Property: dml.HasOne("Email", dml.RelationshipOptions{ForeignKey: true, MappedByUndefined: true})},
})

var Email = dml.Define("email", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "email"}).PrimaryKey()},
})

// endregion: one-sided-has-one

// region: one-sided-belongs-to
// The foreign key column is in the badge table, and Holder has no property
// pointing to Badge.
var Holder = dml.Define("holder", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "hold"}).PrimaryKey()},
})

var Badge = dml.Define("badge", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "badge"}).PrimaryKey()},
	{Name: "holder", Property: dml.BelongsTo("Holder", dml.RelationshipOptions{MappedByUndefined: true})},
})

// endregion: one-sided-belongs-to

// region: many-to-many-pivot
var Order = dml.Define("order_test", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "ord"}).PrimaryKey()},
	{Name: "products", Property: dml.ManyToMany("ProductTest", dml.RelationshipOptions{PivotEntity: "OrdersProducts"})},
})

var ProductTest = dml.Define("product_test", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "prd"}).PrimaryKey()},
	{Name: "orders", Property: dml.ManyToMany("OrderTest")},
})

// OrdersProducts is the pivot table of the relationship, with an extra
// column.
var OrdersProducts = dml.Define("orders_products", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "op"}).PrimaryKey()},
	{Name: "order", Property: dml.BelongsTo("OrderTest", dml.RelationshipOptions{MappedBy: "products"})},
	{Name: "product", Property: dml.BelongsTo("ProductTest", dml.RelationshipOptions{MappedBy: "orders"})},
	{Name: "metadata", Property: dml.JSON().Nullable()},
})

// endregion: many-to-many-pivot

// region: searchable-relationship
var Category = dml.Define("category", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "cat"}).PrimaryKey()},
	{Name: "name", Property: dml.Text()},
	{Name: "entries", Property: dml.HasMany("Entry", dml.RelationshipOptions{MappedBy: "category"}).Searchable()},
})

var Entry = dml.Define("entry", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "entry"}).PrimaryKey()},
	{Name: "title", Property: dml.Text().Searchable()},
	{Name: "category", Property: dml.BelongsTo("Category", dml.RelationshipOptions{MappedBy: "entries"})},
})

// endregion: searchable-relationship
