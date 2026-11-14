package links

// region: imports
import (
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/www/examples/modules/blog"
)

// endregion: imports

// region: define-post-product
// DefinePostProduct links posts and products: a post features many products
// and a product appears in many posts.
func DefinePostProduct() error {
	post := blog.Module.Linkable["post"].ToJSON()
	product := utilsmodulessdk.LinkableOf(&modules.Product)["product"].ToJSON()

	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: post, IsList: true},
		utilsmodulessdk.DefineLinkInput{Linkable: product, IsList: true},
	)
	return err
}

// endregion: define-post-product

// region: define-one-to-one
// DefinePostVariant links a post to one product variant, and a variant to one
// post.
func DefinePostVariant() error {
	post := blog.Module.Linkable["post"]
	variant := utilsmodulessdk.LinkableOf(&modules.Product)["productVariant"]

	_, err := utilsmodulessdk.DefineLink(post, variant)
	return err
}

// endregion: define-one-to-one

// region: define-one-to-many
// DefineCustomerComment links a customer to many comments, while a comment
// belongs to at most one customer.
func DefineCustomerComment() error {
	customer := utilsmodulessdk.LinkableOf(&modules.Customer)["customer"]
	comment := blog.Module.Linkable["comment"].ToJSON()

	_, err := utilsmodulessdk.DefineLink(
		customer,
		utilsmodulessdk.DefineLinkInput{Linkable: comment, IsList: true},
	)
	return err
}

// endregion: define-one-to-many

// region: define-cascade
// DefineSalesChannelPost links a sales channel to many posts, and deletes the
// posts of a sales channel when the channel is deleted.
func DefineSalesChannelPost() error {
	salesChannel := utilsmodulessdk.LinkableOf(&modules.SalesChannel)["salesChannel"]
	post := blog.Module.Linkable["post"].ToJSON()

	_, err := utilsmodulessdk.DefineLink(
		salesChannel,
		utilsmodulessdk.DefineLinkInput{Linkable: post, IsList: true, DeleteCascade: true},
	)
	return err
}

// endregion: define-cascade

// region: define-alias
// DefinePostSalesChannelAlias links posts to sales channels and renames the
// property the linked posts are retrieved with.
func DefinePostSalesChannelAlias() error {
	post := blog.Module.Linkable["post"].ToJSON()
	salesChannel := utilsmodulessdk.LinkableOf(&modules.SalesChannel)["salesChannel"].ToJSON()

	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: post, Alias: "article", IsList: true},
		salesChannel,
	)
	return err
}

// endregion: define-alias

// region: define-all
// Define registers the links of the project. Call it before the
// application is loaded, for example at the start of main.
func Define() error {
	for _, define := range []func() error{
		DefinePostProduct,
		DefinePostSalesChannel,
		DefinePostAuthor,
	} {
		if err := define(); err != nil {
			return err
		}
	}
	return nil
}

// endregion: define-all
