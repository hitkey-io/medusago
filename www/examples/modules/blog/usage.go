package blog

// region: imports
import (
	"context"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// endregion: imports

// region: call-create
// createPosts calls the generated createPosts method.
func createPosts(ctx context.Context, s *BlogModuleService) ([]*dal.Record, error) {
	out, err := s.Call(ctx, "createPosts", []any{
		[]any{
			rec("title", "Hello, world", "slug", "hello-world"),
			rec("title", "Release notes", "slug", "release-notes"),
		},
	})
	if err != nil {
		return nil, err
	}
	return out.([]*dal.Record), nil
}

// endregion: call-create

// region: call-retrieve
// retrievePost calls retrievePost, which returns a NOT_FOUND error when the
// post does not exist.
func retrievePost(ctx context.Context, s *BlogModuleService, id string) (*dal.Record, error) {
	out, err := s.Call(ctx, "retrievePost", []any{
		id,
		rec("select", []any{"id", "title"}, "relations", []any{"comments"}),
	})
	if err != nil {
		return nil, err
	}
	return out.(*dal.Record), nil
}

// endregion: call-retrieve

// region: call-list
// listPosts calls listPosts with filters and a configuration.
func listPosts(ctx context.Context, s *BlogModuleService) ([]*dal.Record, error) {
	out, err := s.Call(ctx, "listPosts",
		[]any{
			rec("status", "published"),
			rec(
				"select", []any{"id", "title"},
				"take", float64(10),
				"skip", float64(0),
				"order", rec("published_at", "DESC"),
			),
		})
	if err != nil {
		return nil, err
	}
	return out.([]*dal.Record), nil
}

// endregion: call-list

// region: call-list-and-count
// listAndCountPosts returns the posts and the total number of matches.
func listAndCountPosts(ctx context.Context, s *BlogModuleService) ([]*dal.Record, int, error) {
	out, err := s.Call(ctx, "listAndCountPosts", []any{
		rec("status", "published"),
		rec("take", float64(10)),
	})
	if err != nil {
		return nil, 0, err
	}
	result := out.([]any)
	return result[0].([]*dal.Record), result[1].(int), nil
}

// endregion: call-list-and-count

// region: call-update
// updatePosts shows the three shapes of the update input.
func updatePosts(ctx context.Context, s *BlogModuleService) error {
	// one object with the primary key: returns the updated record
	if _, err := s.Call(ctx, "updatePosts", []any{
		rec("id", "post_123", "title", "New title"),
	}); err != nil {
		return err
	}

	// a list: returns the updated records
	if _, err := s.Call(ctx, "updatePosts", []any{
		[]any{
			rec("id", "post_123", "title", "New title"),
			rec("id", "post_456", "status", "archived"),
		},
	}); err != nil {
		return err
	}

	// a selector and the data to set on every match
	_, err := s.Call(ctx, "updatePosts", []any{
		rec(
			"selector", rec("status", "draft"),
			"data", rec("status", "archived"),
		),
	})
	return err
}

// endregion: call-update

// region: call-delete
// deletePosts removes posts, or soft-deletes and restores them. The relations
// listed in the cascades of the model follow.
func deletePosts(ctx context.Context, s *BlogModuleService) error {
	if _, err := s.Call(ctx, "deletePosts", []any{[]any{"post_123"}}); err != nil {
		return err
	}

	// the keys of the deleted records, by linkable key, for example
	// { post_id: [...], comment_id: [...] }
	if _, err := s.Call(ctx, "softDeletePosts", []any{[]any{"post_456"}}); err != nil {
		return err
	}
	_, err := s.Call(ctx, "restorePosts", []any{[]any{"post_456"}})
	return err
}

// endregion: call-delete

// region: internal-service
// recentPosts uses the internal service of the Post model, which takes typed
// configuration instead of a record.
func recentPosts(ctx context.Context, s *BlogModuleService) ([]*dal.Record, error) {
	take := 10
	return s.Internal("Post").List(ctx,
		rec("status", "published"),
		utilsmodulessdk.FindConfig{
			Select:    []string{"id", "title"},
			Relations: []string{"comments"},
			Take:      &take,
			Order:     []dal.OrderBy{{Field: "published_at", Direction: "DESC"}},
		},
	)
}

// endregion: internal-service

// region: repository
// postsFromRepository uses the repository of the Post model, which the
// internal service wraps.
func postsFromRepository(ctx context.Context, s *BlogModuleService) ([]*dal.Record, error) {
	take := 5
	return s.Internal("Post").Repository.Find(ctx, dal.FindOptions{
		Where:  rec("status", "published"),
		Select: []string{"id", "title"},
		Order:  []dal.OrderBy{{Field: "published_at", Direction: "DESC"}},
		Take:   &take,
	})
}

// endregion: repository

// region: create-related
// createRelated creates a post with its comments, then a comment of another
// post through its foreign key.
func createRelated(ctx context.Context, s *BlogModuleService) error {
	_, err := s.Call(ctx, "createPosts", []any{
		rec(
			"title", "Hello, world",
			"slug", "hello-world",
			"comments", []any{
				rec("author_name", "Ada", "body", "Great post"),
				rec("author_name", "Grace", "body", "Thanks for sharing"),
			},
		),
	})
	if err != nil {
		return err
	}

	_, err = s.Call(ctx, "createComments", []any{
		rec("author_name", "Linus", "body", "Nice", "post_id", "post_123"),
	})
	return err
}

// endregion: create-related

// region: retrieve-related
// postWithRelations retrieves a post with its comments and tags.
func postWithRelations(ctx context.Context, s *BlogModuleService, id string) (*dal.Record, error) {
	post, err := s.Call(ctx, "retrievePost", []any{
		id,
		rec("relations", []any{"comments", "tags"}),
	})
	if err != nil {
		return nil, err
	}
	return post.(*dal.Record), nil
}

// endregion: retrieve-related

// region: call-signature
// callListPosts shows the shape of every call to a generated method: the name
// of the method, and its arguments as a list.
func callListPosts(ctx context.Context, s *BlogModuleService, filters, config *dal.Record) (any, error) {
	return s.Call(ctx, "listPosts", []any{filters, config})
}

// endregion: call-signature

// region: call-search
// searchPosts uses the q filter, which searches the searchable properties.
func searchPosts(ctx context.Context, s *BlogModuleService) (any, error) {
	return s.Call(ctx, "listPosts", []any{rec("q", "medusa")})
}

// endregion: call-search

// region: replace-many-to-many
// replaceTags relates a post to the tags with the given IDs, and removes its
// other tags.
func replaceTags(ctx context.Context, s *BlogModuleService) error {
	return s.Internal("Post").ReplaceManyToMany(ctx, "post_123", "tags", []any{"tag_1", "tag_2"})
}

// endregion: replace-many-to-many
