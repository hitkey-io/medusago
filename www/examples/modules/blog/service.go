package blog

// region: imports
import (
	"context"
	"fmt"

	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// endregion: imports

// region: service
// BlogModuleService is the service of the blog module. Embedding
// *MedusaService gives it the generated methods (listPosts, createPosts, ...)
// and the internal service of each model.
type BlogModuleService struct {
	*utilsmodulessdk.MedusaService
	options Options
}

// endregion: service

// region: create-draft
// CreateDraft creates a draft post. The longest accepted title comes from the
// module options.
func (s *BlogModuleService) CreateDraft(ctx context.Context, title, slug string) (*dal.Record, error) {
	if len(title) > s.options.MaxTitleLength {
		return nil, fmt.Errorf("the title is longer than %d characters", s.options.MaxTitleLength)
	}
	created, err := s.Internal("Post").Create(ctx, []*dal.Record{rec("title", title, "slug", slug)})
	if err != nil {
		return nil, err
	}
	return created[0], nil
}

// endregion: create-draft

// region: publish
// PublishPost marks a draft post as published.
func (s *BlogModuleService) PublishPost(ctx context.Context, id string) (*dal.Record, error) {
	posts := s.Internal("Post")

	post, err := posts.Retrieve(ctx, id, utilsmodulessdk.FindConfig{Select: []string{"id", "status"}})
	if err != nil {
		return nil, err
	}
	if status, _ := post.Get("status"); status != "draft" {
		return nil, fmt.Errorf("post %s is not a draft", id)
	}

	updated, err := posts.Update(ctx, rec(
		"id", id,
		"status", "published",
		"published_at", dal.FormatJSDate(dal.Now()),
	))
	if err != nil {
		return nil, err
	}
	return updated.(*dal.Record), nil
}

// endregion: publish

// region: raw-sql
// IncrementViews adds one view to a post with a single SQL statement and
// returns the new count.
func (s *BlogModuleService) IncrementViews(ctx context.Context, id string) (float64, error) {
	rows, err := dal.QueryRows(ctx, s.DB(),
		`update "post" set "views" = "views" + 1 where "id" = $1 and "deleted_at" is null returning "views"`, id)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, fmt.Errorf("post %s was not found", id)
	}
	return rows[0]["views"].(float64), nil
}

// endregion: raw-sql

// region: transaction
// SetPostTags replaces the tags of a post with the given values, creating
// the tags that do not exist yet. Everything runs in one transaction.
func (s *BlogModuleService) SetPostTags(ctx context.Context, postID string, values []string) error {
	return s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		tags := tx.Internal("Tag")

		wanted := make([]any, len(values))
		for i, v := range values {
			wanted[i] = v
		}
		existing, err := tags.List(ctx, rec("value", rec("$in", wanted)), utilsmodulessdk.FindConfig{})
		if err != nil {
			return err
		}

		ids := map[string]any{}
		for _, tag := range existing {
			value, _ := tag.Get("value")
			id, _ := tag.Get("id")
			ids[value.(string)] = id
		}

		var missing []*dal.Record
		for _, v := range values {
			if _, ok := ids[v]; !ok {
				ids[v] = nil
				missing = append(missing, rec("value", v))
			}
		}
		created, err := tags.Create(ctx, missing)
		if err != nil {
			return err
		}
		for _, tag := range created {
			value, _ := tag.Get("value")
			ids[value.(string)], _ = tag.Get("id")
		}

		tagIDs := make([]any, 0, len(ids))
		for _, id := range ids {
			tagIDs = append(tagIDs, id)
		}
		return tx.Internal("Post").ReplaceManyToMany(ctx, postID, "tags", tagIDs)
	})
}

// endregion: transaction

// region: transaction-raw
// MoveComments moves the comments of a post to another post, and returns how
// many it moved. The target post is locked for the duration of the
// transaction, so it can't be deleted while its comments move.
func (s *BlogModuleService) MoveComments(ctx context.Context, fromPostID, toPostID string) (int, error) {
	moved := 0
	err := s.InTransaction(ctx, func(tx *utilsmodulessdk.MedusaService) error {
		posts, err := dal.QueryRows(ctx, tx.DB(),
			`select "id" from "post" where "id" = $1 and "deleted_at" is null for update`, toPostID)
		if err != nil {
			return err
		}
		if len(posts) == 0 {
			return fmt.Errorf("post %s was not found", toPostID)
		}

		rows, err := dal.QueryRows(ctx, tx.DB(),
			`update "comment" set "post_id" = $1 where "post_id" = $2 and "deleted_at" is null returning "id"`,
			toPostID, fromPostID)
		if err != nil {
			return err
		}
		moved = len(rows)
		return nil
	})
	return moved, err
}

// endregion: transaction-raw
