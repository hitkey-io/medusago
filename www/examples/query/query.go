package queries

// region: imports
import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/caching"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
)

// endregion: imports

// region: resolver
// Resolver is what routes (req.Scope), workflow steps and scripts resolve
// from: the container.
type Resolver interface {
	Resolve(key string) any
}

func queryOf(c Resolver) *query.Query {
	return c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
}

// endregion: resolver

// region: rec
// rec builds a *dal.Record, an ordered map that encodes to JSON with its
// keys in insertion order.
func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// endregion: rec

// region: graph
// PostTitles returns the ID and title of every post.
func PostTitles(ctx context.Context, c Resolver) ([]string, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{"id", "title"},
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}

	titles := make([]string, 0, len(res.Data))
	for _, row := range res.Data {
		post := row.(*dal.Record)
		title, _ := post.Get("title")
		titles = append(titles, title.(string))
	}
	return titles, nil
}

// endregion: graph

// region: all-fields
// PostWithAllFields retrieves every field of one post.
func PostWithAllFields(ctx context.Context, c Resolver, id string) (*dal.Record, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity:  "post",
		Fields:  []string{"*"},
		Filters: rec("id", id),
	}, joiner.Options{})
	if err != nil || len(res.Data) == 0 {
		return nil, err
	}
	return res.Data[0].(*dal.Record), nil
}

// endregion: all-fields

// region: linked
// PostsWithLinkedRecords retrieves posts with their comments (a relationship
// of the blog module) and their linked products and sales channels (module
// links).
func PostsWithLinkedRecords(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{
			"id",
			"title",
			"comments.id",
			"comments.body",
			"products.*",
			"products.variants.*",
		},
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: linked

// region: filters
// PublishedPosts retrieves the published posts of an author.
func PublishedPosts(ctx context.Context, c Resolver, authorID string) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{"id", "title"},
		Filters: rec(
			"status", "published",
			"author_id", authorID,
		),
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: filters

// region: filter-in
func filterIn() *dal.Record {
	return rec("id", []any{"post_123", "post_456"})
}

// endregion: filter-in

// region: filter-operators
func filterOperators() []*dal.Record {
	return []*dal.Record{
		// not equal
		rec("status", rec("$ne", "draft")),
		// not in a list of values
		rec("status", rec("$nin", []any{"draft", "archived"})),
		// a range
		rec("views", rec("$gte", float64(100), "$lt", float64(1000))),
		// like (case-insensitive: $ilike)
		rec("title", rec("$ilike", "%medusa%")),
		// is not null
		rec("published_at", rec("$ne", nil)),
		// or
		rec("$or", []any{
			rec("status", "published"),
			rec("views", rec("$gt", float64(1000))),
		}),
	}
}

// endregion: filter-operators

// region: filter-relation
// PostsByCommenter retrieves the posts that have a comment by Ada.
func PostsByCommenter(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity:  "post",
		Fields:  []string{"id", "title", "comments.*"},
		Filters: rec("comments", rec("author_name", "Ada")),
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: filter-relation

// region: filter-linked
// PostsOfProduct retrieves the posts linked to the product with a handle.
func PostsOfProduct(ctx context.Context, c Resolver, handle string) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity:  "post",
		Fields:  []string{"id", "title", "products.*"},
		Filters: rec("products", rec("handle", handle)),
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: filter-linked

// region: pagination
// Page is one page of posts and the total number of posts.
type Page struct {
	Posts []any
	Count int
}

// PostsPage retrieves a page of posts, the newest first.
func PostsPage(ctx context.Context, c Resolver, offset, limit int) (*Page, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{"id", "title"},
		Pagination: rec(
			"skip", float64(offset),
			"take", float64(limit),
			"order", rec("created_at", "DESC"),
		),
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}

	// the metadata is set when skip is passed
	count, _ := res.Metadata.Get("count")
	return &Page{Posts: res.Data, Count: int(count.(float64))}, nil
}

// endregion: pagination

// region: count-only
// CountPosts returns the number of posts without retrieving them.
func CountPosts(ctx context.Context, c Resolver) (int, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity:     "post",
		Fields:     []string{"id"},
		Pagination: rec("skip", float64(0), "take", float64(0)),
	}, joiner.Options{})
	if err != nil {
		return 0, err
	}
	count, _ := res.Metadata.Get("count")
	return int(count.(float64)), nil
}

// endregion: count-only

// region: with-deleted
// DeletedPosts retrieves only the posts that were soft-deleted.
func DeletedPosts(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity:      "post",
		Fields:      []string{"id", "title", "deleted_at"},
		Filters:     rec("deleted_at", rec("$ne", nil)),
		WithDeleted: true,
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: with-deleted

// region: throw
// RetrievePost returns a NOT_FOUND error when no post has the ID, instead of
// an empty result.
func RetrievePost(ctx context.Context, c Resolver, id string) (*dal.Record, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity:  "post",
		Fields:  []string{"id", "title"},
		Filters: rec("id", id),
	}, joiner.Options{ThrowIfKeyNotFound: true})
	if err != nil {
		return nil, err
	}
	return res.Data[0].(*dal.Record), nil
}

// endregion: throw

// region: read-only
// PostsWithAuthors retrieves posts with their authors, through the read-only
// link between post.author_id and the user module.
func PostsWithAuthors(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{"id", "title", "user.email", "user.first_name"},
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: read-only

// region: read-only-inverse
// UsersWithPosts retrieves users with the posts they wrote, through the
// inverse read-only link.
func UsersWithPosts(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "user",
		Fields: []string{"id", "email", "posts.id", "posts.title"},
	}, joiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: read-only-inverse

// region: locale
// LocalizedPosts retrieves the posts translated for a locale. It needs the
// Translation Module.
func LocalizedPosts(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{"id", "title"},
	}, joiner.Options{Locale: "fr-FR"})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: locale

// region: cache
// CachedPosts caches the result of the query. It needs the caching module,
// and the caching feature flag.
func CachedPosts(ctx context.Context, c Resolver) ([]any, error) {
	ttl := 60
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{"id", "title"},
	}, joiner.Options{Cache: &caching.CacheOptions{
		Key:                  "published-posts",
		Tags:                 []string{"posts"},
		TTL:                  &ttl,
		ComputeAutomaticTags: true,
	}})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: cache
