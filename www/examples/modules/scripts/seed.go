package scripts

// region: seed-imports
import (
	"context"
	"fmt"
	"log/slog"

	saleschannelworkflows "github.com/hitkey-io/medusago/packages/core/core-flows/src/sales-channel/workflows"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	typescommon "github.com/hitkey-io/medusago/packages/core/types/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// endregion: seed-imports

func rec(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// region: seed
// Seed creates a sales channel for the blog, three posts, and links the
// posts to the channel. It is registered with the name "seed".
func Seed(ctx context.Context, args typescommon.ExecArgs) error {
	c := args.Container
	logger := c.Resolve(common.ContainerRegistrationKeys.LOGGER).(*slog.Logger)
	q := c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
	link := c.Resolve(common.ContainerRegistrationKeys.LINK).(*modulessdk.Link)
	blogService := c.Resolve(blog.Name).(*blog.BlogModuleService)

	logger.Info("Seeding blog data...")

	// Do nothing if a previous run already created the posts.
	seeded, err := blogService.Internal("Post").List(ctx, rec("slug", "hello-world"), utilsmodulessdk.FindConfig{})
	if err != nil {
		return err
	}
	if len(seeded) > 0 {
		logger.Info("The blog data is already seeded")
		return nil
	}

	// Reuse the sales channel if it exists.
	existing, err := q.Graph(ctx, query.GraphInput{
		Entity:  "sales_channel",
		Fields:  []string{"id"},
		Filters: rec("name", "Blog"),
	}, joiner.Options{})
	if err != nil {
		return err
	}
	var channelID any
	if len(existing.Data) > 0 {
		channelID, _ = existing.Data[0].(*dal.Record).Get("id")
	} else {
		channels, err := saleschannelworkflows.CreateSalesChannelsWorkflow.Run(ctx, c, saleschannelworkflows.CreateSalesChannelsInput{
			SalesChannelsData: []*dal.Record{rec("name", "Blog")},
		})
		if err != nil {
			return err
		}
		channelID, _ = channels[0].Get("id")
	}

	posts, err := blogService.Internal("Post").Create(ctx, []*dal.Record{
		rec("title", "Hello, world", "slug", "hello-world", "status", "published"),
		rec("title", "Release notes", "slug", "release-notes", "status", "published"),
		rec("title", "Work in progress", "slug", "work-in-progress"),
	})
	if err != nil {
		return err
	}

	links := make([]modulessdk.LinkDefinition, len(posts))
	for i, post := range posts {
		postID, _ := post.Get("id")
		links[i] = modulessdk.NewLinkDefinition(
			blog.Name, "post_id", postID,
			utilsmodulessdk.SALES_CHANNEL, "sales_channel_id", channelID,
		)
		links[i].Set("data", rec("position", float64(i)))
	}
	if _, err := link.Create(ctx, links); err != nil {
		return err
	}

	logger.Info(fmt.Sprintf("Seeded %d posts", len(posts)))
	return nil
}

// endregion: seed
