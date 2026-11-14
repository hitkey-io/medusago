package links

// region: imports
import (
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// endregion: imports

// region: define-read-only
// DefinePostAuthor lets a post retrieve its author, the admin user whose ID
// is stored in post.author_id.
func DefinePostAuthor() error {
	post := blog.Module.Linkable["post"].ToJSON()
	user := utilsmodulessdk.LinkableOf(&modules.User)["user"].ToJSON()

	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: post, Field: "author_id"},
		user,
		utilsmodulessdk.DefineLinkOptions{ReadOnly: true},
	)
	return err
}

// endregion: define-read-only

// region: define-read-only-list
// DefineAuthorPosts is the inverse of DefinePostAuthor: a user retrieves the
// posts whose author_id is the user's ID. IsList makes it a list, as an author
// has many posts.
func DefineAuthorPosts() error {
	user := utilsmodulessdk.LinkableOf(&modules.User)["user"].ToJSON()
	post := blog.Module.Linkable["post"].ToJSON()
	// the field of the post that holds the user's ID
	post.PrimaryKey = "author_id"

	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: user, Field: "id", IsList: true},
		post,
		utilsmodulessdk.DefineLinkOptions{ReadOnly: true},
	)
	return err
}

// endregion: define-read-only-list
