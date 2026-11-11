package dbgenerate

import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// The Go versions of the models of versions/v*/ (each version returns new
// entities).

func brandProfile() *dml.Entity {
	return dml.Define("brand_profile", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "bprof"}).PrimaryKey()},
		{Name: "website", Property: dml.Text().Nullable()},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "profile"})},
	})
}

func v1() []*dml.Entity {
	brand := dml.Define("brand", dml.Schema{
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
	logo := dml.Define("brand_logo", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "blogo"}).PrimaryKey()},
		{Name: "url", Property: dml.Text()},
		{Name: "position", Property: dml.Number().Default(0)},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "logos"})},
	})
	stat := dml.Define("brand_stat", dml.Schema{
		{Name: "brand_id", Property: dml.Text().PrimaryKey()},
		{Name: "period", Property: dml.Text().PrimaryKey()},
		{Name: "views", Property: dml.Number().Default(0)},
		{Name: "score", Property: dml.Float().Nullable()},
	})
	return []*dml.Entity{logo, brandProfile(), stat, brand}
}

func v2() []*dml.Entity {
	brand := dml.Define("brand", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "brand"}).PrimaryKey()},
		{Name: "name", Property: dml.Text().Searchable()},
		{Name: "handle", Property: dml.Text().Unique()},
		{Name: "description", Property: dml.Text()},
		{Name: "slogan", Property: dml.Text().Index().Nullable()},
		{Name: "is_active", Property: dml.Boolean().Default(true)},
		{Name: "rank", Property: dml.Number().Default(10)},
		{Name: "rating", Property: dml.Float().Default(0)},
		{Name: "revenue", Property: dml.BigNumber().Nullable()},
		{Name: "budget", Property: dml.BigNumber().Default(100)},
		{Name: "status", Property: dml.Enum("draft", "published", "archived", "retired").Default("draft")},
		{Name: "country", Property: dml.Enum("us", "de").Nullable()},
		{Name: "tags", Property: dml.Array().Default([]any{"new"})},
		{Name: "metadata", Property: dml.JSON().Nullable()},
		{Name: "settings", Property: dml.JSON().Default(map[string]any{"theme": "dark"})},
		{Name: "launched_at", Property: dml.DateTime().Nullable()},
		{Name: "founded_year", Property: dml.Number().Nullable()},
		{Name: "sequence", Property: dml.Autoincrement()},
		{Name: "logos", Property: dml.HasMany("BrandLogo", dml.RelationshipOptions{MappedBy: "brand"})},
		{Name: "profile", Property: dml.HasOne("BrandProfile", dml.RelationshipOptions{MappedBy: "brand"}).Nullable()},
		{Name: "regions", Property: dml.HasMany("BrandRegion", dml.RelationshipOptions{MappedBy: "brand"})},
		{Name: "labels", Property: dml.ManyToMany("BrandLabel", dml.RelationshipOptions{MappedBy: "brands"})},
	}).
		WithIndexes([]dml.Index{
			{On: []string{"name", "status"}, Where: "deleted_at IS NULL"},
			{On: []string{"handle", "country"}, Unique: true},
			{Name: "IDX_brand_rank_rating", On: []string{"rank", "rating"}, Where: "rank > 0"},
		}).
		WithChecks([]dml.Check{{Name: "brand_rank_check", Expression: "rank >= 0"}, {Expression: "rating >= 0"}}).
		WithCascades(dml.Cascades{Delete: []string{"logos", "profile"}})
	label := dml.Define("brand_label", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "blabel"}).PrimaryKey()},
		{Name: "value", Property: dml.Text()},
		{Name: "brands", Property: dml.ManyToMany("Brand", dml.RelationshipOptions{MappedBy: "labels"})},
	})
	region := dml.Define("brand_region", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "breg"}).PrimaryKey()},
		{Name: "code", Property: dml.Text()},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "regions"}).Nullable()},
	})
	logo := dml.Define("brand_logo", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "blogo"}).PrimaryKey()},
		{Name: "url", Property: dml.Text()},
		{Name: "alt", Property: dml.Text().Nullable()},
		{Name: "position", Property: dml.Float().Default(0)},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "logos"})},
	})
	stat := dml.Define("brand_stat", dml.Schema{
		{Name: "brand_id", Property: dml.Text().PrimaryKey()},
		{Name: "period", Property: dml.Text().PrimaryKey()},
		{Name: "views", Property: dml.Number().Default(0)},
		{Name: "score", Property: dml.Number().Nullable()},
		{Name: "notes", Property: dml.Text().Nullable()},
	})
	return []*dml.Entity{label, logo, brandProfile(), region, stat, brand}
}

// v3Brand is the brand model of versions 3 to 5 (extra adds the properties
// of version 5 before the relationships).
func v3Brand(extra dml.Schema, relations dml.Schema) *dml.Entity {
	schema := dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "brand"}).PrimaryKey()},
		{Name: "name", Property: dml.Text().Searchable()},
		{Name: "handle", Property: dml.Text().Unique()},
		{Name: "description", Property: dml.Text()},
		{Name: "tagline", Property: dml.Text().Index().Nullable()},
	}
	schema = append(schema, extra...)
	schema = append(schema, dml.Schema{
		{Name: "is_active", Property: dml.Boolean().Default(true)},
		{Name: "rank", Property: dml.Number().Default(10)},
		{Name: "rating", Property: dml.Float().Default(0)},
		{Name: "revenue", Property: dml.BigNumber().Nullable()},
		{Name: "budget", Property: dml.BigNumber().Default(100)},
		{Name: "status", Property: dml.Enum("draft", "published", "retired").Default("draft")},
		{Name: "country", Property: dml.Enum("us", "de").Default("us")},
		{Name: "tags", Property: dml.Array().Default([]any{"new"})},
		{Name: "metadata", Property: dml.JSON().Nullable()},
		{Name: "settings", Property: dml.JSON().Default(map[string]any{"theme": "dark"})},
		{Name: "launched_at", Property: dml.DateTime().Nullable()},
		{Name: "founded_year", Property: dml.Text().Nullable()},
		{Name: "sequence", Property: dml.Autoincrement()},
		{Name: "logos", Property: dml.HasMany("BrandLogo", dml.RelationshipOptions{MappedBy: "brand"})},
		{Name: "profile", Property: dml.HasOne("BrandProfile", dml.RelationshipOptions{MappedBy: "brand"}).Nullable()},
		{Name: "regions", Property: dml.HasMany("BrandRegion", dml.RelationshipOptions{MappedBy: "brand"})},
	}...)
	schema = append(schema, relations...)
	return dml.Define("brand", schema).
		WithIndexes([]dml.Index{
			{On: []string{"name", "status"}, Where: "deleted_at IS NULL"},
			{Name: "IDX_brand_rank_rating", On: []string{"rank", "rating"}, Where: "rank > 0"},
		}).
		WithChecks([]dml.Check{{Name: "brand_rank_check", Expression: "rank >= 1"}, {Expression: "rating >= 0"}}).
		WithCascades(dml.Cascades{Delete: []string{"profile"}})
}

func v3Others() []*dml.Entity {
	region := dml.Define("brand_region", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "breg"}).PrimaryKey()},
		{Name: "code", Property: dml.Text()},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "regions"})},
	})
	logo := dml.Define("brand_logo", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "blogo"}).PrimaryKey()},
		{Name: "url", Property: dml.Text()},
		{Name: "alt", Property: dml.Text().Nullable()},
		{Name: "position", Property: dml.Float().Default(0)},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "logos"})},
	})
	stat := dml.Define("brand_stat", dml.Schema{
		{Name: "brand_id", Property: dml.Text().PrimaryKey()},
		{Name: "period", Property: dml.Text().PrimaryKey()},
		{Name: "region", Property: dml.Text().PrimaryKey()},
		{Name: "views", Property: dml.Number().Default(0)},
		{Name: "score", Property: dml.Number().Nullable()},
		{Name: "notes", Property: dml.Text().Nullable()},
	})
	return []*dml.Entity{logo, brandProfile(), region, stat}
}

func v3() []*dml.Entity {
	return append(v3Others(), v3Brand(nil, nil))
}

func v4() []*dml.Entity { return v3() }

func v5() []*dml.Entity {
	note := dml.Define("brand_note", dml.Schema{
		{Name: "id", Property: dml.Autoincrement().PrimaryKey()},
		{Name: "body", Property: dml.Text()},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "notes"})},
	})
	brand := v3Brand(
		dml.Schema{{Name: "code", Property: dml.Text().Unique("IDX_brand_code_custom").Nullable()}},
		dml.Schema{
			{Name: "notes", Property: dml.HasMany("BrandNote", dml.RelationshipOptions{MappedBy: "brand"})},
			{Name: "featured_logo", Property: dml.HasOne("BrandLogo", dml.RelationshipOptions{ForeignKey: true, MappedByUndefined: true}).Nullable()},
		},
	)
	return append(append(v3Others(), note), brand)
}

func v6() []*dml.Entity {
	brand := dml.Define("brand", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "brand"}).PrimaryKey()},
		{Name: "name", Property: dml.Text().Searchable()},
		{Name: "handle", Property: dml.Text().Unique()},
		{Name: "description", Property: dml.Text()},
		{Name: "tagline", Property: dml.Text().Index().Nullable()},
		{Name: "code", Property: dml.Text().Unique("IDX_brand_code_custom").Nullable()},
		{Name: "is_active", Property: dml.Boolean().Default(true)},
		{Name: "rank", Property: dml.Number().Default(10)},
		{Name: "rating", Property: dml.Float().Default(0)},
		{Name: "revenue", Property: dml.BigNumber().Nullable()},
		{Name: "budget", Property: dml.BigNumber().Default(100)},
		{Name: "status", Property: dml.Enum("draft", "published", "retired").Default("draft")},
		{Name: "country", Property: dml.Enum("us", "de").Default("us")},
		{Name: "tags", Property: dml.Array().Default([]any{"new"})},
		{Name: "metadata", Property: dml.JSON().Nullable()},
		{Name: "settings", Property: dml.JSON().Default(map[string]any{"theme": "dark"})},
		{Name: "launched_at", Property: dml.DateTime().Nullable()},
		{Name: "founded_year", Property: dml.Text().Nullable()},
		{Name: "sequence", Property: dml.Autoincrement()},
		{Name: "profile", Property: dml.HasOne("BrandProfile", dml.RelationshipOptions{MappedBy: "brand"}).Nullable()},
		{Name: "regions", Property: dml.HasMany("BrandRegion", dml.RelationshipOptions{MappedBy: "brand"})},
		{Name: "notes", Property: dml.HasMany("BrandNote", dml.RelationshipOptions{MappedBy: "brand"})},
		{Name: "colors", Property: dml.HasMany("BrandColor", dml.RelationshipOptions{MappedBy: "brand"})},
		{Name: "brand_tags", Property: dml.ManyToMany("BrandTag", dml.RelationshipOptions{
			MappedBy:          "brands",
			PivotTable:        "brand_tag_links",
			JoinColumn:        "brand_ref_id",
			InverseJoinColumn: "tag_ref_id",
		})},
		{Name: "owner", Property: dml.HasOne("BrandOwner", dml.RelationshipOptions{MappedByUndefined: true}).Nullable()},
	}).
		WithIndexes([]dml.Index{
			{On: []string{"name", "status"}, Where: "deleted_at IS NULL"},
			{Name: "IDX_brand_rank_rating", On: []string{"rank", "rating"}, Where: "rank > 0"},
		}).
		WithChecks([]dml.Check{{Name: "brand_rank_check", Expression: "rank >= 1"}, {Expression: "rating >= 0"}}).
		WithCascades(dml.Cascades{Delete: []string{"profile", "colors"}})
	color := dml.Define("brand_color", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "bcol"}).PrimaryKey()},
		{Name: "hex", Property: dml.Text().Default("#000000")},
		{Name: "tone", Property: dml.Enum("light", "dark").Default("light")},
		{Name: "ratio", Property: dml.Float().Default(1.5)},
		{Name: "weight", Property: dml.Number().Index("IDX_brand_color_weight_custom").Nullable()},
		{Name: "is_primary", Property: dml.Boolean().Default(false)},
		{Name: "palette", Property: dml.JSON().Default([]any{"red", "blue"})},
		{Name: "shades", Property: dml.Array().Nullable()},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "colors"})},
	}).
		WithIndexes([]dml.Index{
			{On: []string{"tone"}, Type: "brin"},
			{Name: "IDX_brand_color_ratio_all", On: []string{"ratio"}, WhereNull: true},
			{On: []string{"hex", "tone"}, Where: "is_primary = true"},
		})
	tag := dml.Define("brand_tag", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "btag"}).PrimaryKey()},
		{Name: "value", Property: dml.Text()},
		{Name: "brands", Property: dml.ManyToMany("Brand", dml.RelationshipOptions{MappedBy: "brand_tags"})},
	})
	owner := dml.Define("brand_owner", dml.Schema{
		{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "bown"}).PrimaryKey()},
		{Name: "name", Property: dml.Text()},
	})
	note := dml.Define("brand_note", dml.Schema{
		{Name: "id", Property: dml.Autoincrement().PrimaryKey()},
		{Name: "body", Property: dml.Text()},
		{Name: "brand", Property: dml.BelongsTo("Brand", dml.RelationshipOptions{MappedBy: "notes"})},
	})
	others := v3Others()
	// v3Others: logo, profile, region, stat; the logo model is removed
	return []*dml.Entity{color, note, owner, others[1], others[2], others[3], tag, brand}
}

// versions are the model versions, in order.
var versions = []struct {
	name   string
	models func() []*dml.Entity
}{
	{"v1", v1},
	{"v2", v2},
	{"v3", v3},
	{"v4", v4},
	{"v5", v5},
	{"v6", v6},
}
