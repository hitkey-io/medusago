// Package brand is the Go version of the brand test module of the http-diff
// harness (testapp/src/modules/brand): a custom module with DML models, its
// links to core modules and its API routes.
package brand

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// Brand mirrors models/brand.js.
var Brand = dml.Define("brand", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "brand"}).PrimaryKey()},
	{Name: "name", Property: dml.Text().Searchable()},
	{Name: "handle", Property: dml.Text().Unique()},
	{Name: "description", Property: dml.Text().Nullable()},
	{Name: "is_active", Property: dml.Boolean().Default(true)},
	{Name: "rank", Property: dml.Number().Default(0)},
	{Name: "rating", Property: dml.Float().Nullable()},
	{Name: "revenue", Property: dml.BigNumber().Nullable()},
	{Name: "status", Property: dml.Enum("draft", "published", "archived").Default("draft")},
	{Name: "tags", Property: dml.Array().Default([]any{})},
	{Name: "currency_code", Property: dml.Text().Nullable()},
	{Name: "metadata", Property: dml.JSON().Nullable()},
	{Name: "launched_at", Property: dml.DateTime().Nullable()},
	{Name: "logos", Property: dml.HasMany("BrandLogo", dml.RelationshipOptions{MappedBy: "brand"})},
	{Name: "profile", Property: dml.HasOne("BrandProfile", dml.RelationshipOptions{MappedBy: "brand"}).Nullable()},
}).
	WithIndexes([]dml.Index{{On: []string{"name", "status"}, Where: "deleted_at IS NULL"}}).
	WithCascades(dml.Cascades{Delete: []string{"logos", "profile"}})

// BrandLogo mirrors models/brand-logo.js.
var BrandLogo = dml.Define("brand_logo", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "blogo"}).PrimaryKey()},
	{Name: "url", Property: dml.Text()},
	{Name: "position", Property: dml.Number().Default(0)},
	{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "logos"})},
})

// BrandProfile mirrors models/brand-profile.js.
var BrandProfile = dml.Define("brand_profile", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "bprof"}).PrimaryKey()},
	{Name: "website", Property: dml.Text().Nullable()},
	{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "profile"})},
})

// BrandStat mirrors models/brand-stat.js (composite primary key).
var BrandStat = dml.Define("brand_stat", dml.Schema{
	{Name: "brand_id", Property: dml.Text().PrimaryKey()},
	{Name: "period", Property: dml.Text().PrimaryKey()},
	{Name: "views", Property: dml.Number().Default(0)},
	{Name: "score", Property: dml.Float().Nullable()},
})

// Models are the MedusaService model objects, in service.js order.
var Models = []*dml.Entity{Brand, BrandLogo, BrandProfile, BrandStat}
