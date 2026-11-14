package scripts

// region: imports
import (
	"context"
	"fmt"
	"log/slog"

	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// endregion: imports

// region: test-module
// TestBlog creates a draft with the blog module and lists the posts.
func TestBlog(ctx context.Context, args typescommon.ExecArgs) error {
	logger := args.Container.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)

	// the module service is registered under the key of the module
	blogService := args.Container.Resolve(blog.Name).(*blog.BlogModuleService)

	if _, err := blogService.CreateDraft(ctx, "My first post", "my-first-post"); err != nil {
		return err
	}

	posts, err := blogService.Internal("Post").List(ctx, dal.NewRecord(), utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	for _, post := range posts {
		title, _ := post.Get("title")
		status, _ := post.Get("status")
		logger.Info(fmt.Sprintf("%v (%v)", title, status))
	}
	return nil
}

// endregion: test-module
