package links

// region: imports
import (
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// endregion: imports

func strPtr(s string) *string { return &s }

// region: define-custom-columns
// DefinePostSalesChannel links posts to sales channels with extra columns in
// the link table.
func DefinePostSalesChannel() error {
	post := blog.Module.Linkable["post"].ToJSON()
	salesChannel := utilsmodulessdk.LinkableOf(&modules.SalesChannel)["salesChannel"].ToJSON()

	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: post, IsList: true},
		utilsmodulessdk.DefineLinkInput{Linkable: salesChannel, IsList: true},
		utilsmodulessdk.DefineLinkOptions{Database: &utilsmodulessdk.LinkDatabaseOptions{
			Table: "post_sales_channel",
			ExtraColumns: joiner.ExtraFields{
				{Name: "position", ExtraField: joiner.ExtraField{Type: "integer", DefaultValue: strPtr("0")}},
				{Name: "pinned", ExtraField: joiner.ExtraField{Type: "boolean", DefaultValue: strPtr("false")}},
				{Name: "note", ExtraField: joiner.ExtraField{Type: "text", Nullable: true}},
			},
		}},
	)
	return err
}

// endregion: define-custom-columns
