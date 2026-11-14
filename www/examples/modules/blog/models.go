package blog

// region: imports
import "github.com/hitkey-io/medusago/packages/core/utils/src/dml"

// endregion: imports

// region: post
var Post = dml.Define("post", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "post"}).PrimaryKey()},
	{Name: "title", Property: dml.Text().Searchable()},
	{Name: "slug", Property: dml.Text().Unique()},
	{Name: "content", Property: dml.Text().Nullable()},
	{Name: "status", Property: dml.Enum("draft", "published", "archived").Default("draft")},
	{Name: "views", Property: dml.Number().Default(0)},
	{Name: "published_at", Property: dml.DateTime().Nullable()},
	{Name: "author_id", Property: dml.Text().Nullable()},
	{Name: "metadata", Property: dml.JSON().Nullable()},
	{Name: "comments", Property: dml.HasMany("Comment", dml.RelationshipOptions{MappedBy: "post"})},
	{Name: "tags", Property: dml.ManyToMany("Tag", dml.RelationshipOptions{MappedBy: "posts"})},
}).
	WithIndexes([]dml.Index{{On: []string{"status", "published_at"}}}).
	WithCascades(dml.Cascades{Delete: []string{"comments"}})

// endregion: post

// region: comment
var Comment = dml.Define("comment", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "cmt"}).PrimaryKey()},
	{Name: "author_name", Property: dml.Text()},
	{Name: "body", Property: dml.Text()},
	{Name: "post", Property: dml.BelongsTo("Post", dml.RelationshipOptions{MappedBy: "comments"})},
})

// endregion: comment

// region: tag
var Tag = dml.Define("tag", dml.Schema{
	{Name: "id", Property: dml.Id(dml.IdOptions{Prefix: "tag"}).PrimaryKey()},
	{Name: "value", Property: dml.Text().Unique()},
	{Name: "posts", Property: dml.ManyToMany("Post", dml.RelationshipOptions{MappedBy: "tags"})},
})

// endregion: tag

// region: models
// Models lists the data models of the module.
var Models = []*dml.Entity{Post, Comment, Tag}

// endregion: models
