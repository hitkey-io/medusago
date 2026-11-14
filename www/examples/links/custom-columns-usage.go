package links

// region: imports
import (
	"context"

	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	queryjoiner "github.com/hitkey-io/medusago/packages/core/query/src/joiner"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
)

// endregion: imports

func data(kv ...any) *dal.Record {
	r := dal.NewRecord()
	for i := 0; i+1 < len(kv); i += 2 {
		r.Set(kv[i].(string), kv[i+1])
	}
	return r
}

// region: create-with-data
// LinkPostToSalesChannel links a post to a sales channel and sets the custom
// columns of the link.
func LinkPostToSalesChannel(ctx context.Context, c Resolver, postID, salesChannelID string) error {
	link := modulessdk.NewLinkDefinition(
		"blog", "post_id", postID,
		utilsmodulessdk.SALES_CHANNEL, "sales_channel_id", salesChannelID,
	)
	link.Set("data", data("position", float64(1), "pinned", true))

	_, err := linkOf(c).Create(ctx, []modulessdk.LinkDefinition{link})
	return err
}

// endregion: create-with-data

// region: update-data
// PinPost updates the custom columns of an existing link: Create updates the
// link when it exists. It passes every column, as the ones it leaves out are
// reset to their default values.
func PinPost(ctx context.Context, c Resolver, postID, salesChannelID string) error {
	link := modulessdk.NewLinkDefinition(
		"blog", "post_id", postID,
		utilsmodulessdk.SALES_CHANNEL, "sales_channel_id", salesChannelID,
	)
	link.Set("data", data("position", float64(1), "pinned", true, "note", "Featured on the homepage"))

	_, err := linkOf(c).Create(ctx, []modulessdk.LinkDefinition{link})
	return err
}

// endregion: update-data

// region: query-special-field
// PostsWithLinkRows retrieves posts with the rows of their sales channel
// links, custom columns included, through the special link field.
func PostsWithLinkRows(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity: "post",
		Fields: []string{"id", "title", "sales_channel_link.*", "sales_channels.*"},
	}, queryjoiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: query-special-field

// region: query-entry-point
// PinnedPostLinks retrieves the pinned links with their post and sales
// channel through the entry point of the link.
func PinnedPostLinks(ctx context.Context, c Resolver) ([]any, error) {
	res, err := queryOf(c).Graph(ctx, query.GraphInput{
		Entity:  "post_sales_channel",
		Fields:  []string{"position", "pinned", "note", "post.*", "sales_channel.*"},
		Filters: data("pinned", true),
	}, queryjoiner.Options{})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// endregion: query-entry-point
