package links

// region: imports
import (
	"context"

	commonsteps "github.com/hitkey-io/medusago/packages/core/core-flows/src/common/steps"
	modulessdk "github.com/hitkey-io/medusago/packages/core/modules-sdk/src"
	"github.com/hitkey-io/medusago/packages/core/query/src/query"
	"github.com/hitkey-io/medusago/packages/core/utils/src/common"
	"github.com/hitkey-io/medusago/packages/core/utils/src/dal"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/core/workflows-sdk/src/utils/composer"
)

// endregion: imports

// region: resolver
// Resolver is what routes (req.Scope), workflow steps and scripts resolve
// from: the container.
type Resolver interface {
	Resolve(key string) any
}

func linkOf(c Resolver) *modulessdk.Link {
	return c.Resolve(common.ContainerRegistrationKeys.LINK).(*modulessdk.Link)
}

func queryOf(c Resolver) *query.Query {
	return c.Resolve(common.ContainerRegistrationKeys.QUERY).(*query.Query)
}

// endregion: resolver

// region: create
// LinkPostToProduct links a post to a product.
func LinkPostToProduct(ctx context.Context, c Resolver, postID, productID string) error {
	_, err := linkOf(c).Create(ctx, []modulessdk.LinkDefinition{
		// the modules and their keys, in the order the link is defined
		modulessdk.NewLinkDefinition(
			"blog", "post_id", postID,
			utilsmodulessdk.PRODUCT, "product_id", productID,
		),
	})
	return err
}

// endregion: create

// region: dismiss
// UnlinkPostFromProduct removes the link between a post and a product.
func UnlinkPostFromProduct(ctx context.Context, c Resolver, postID, productID string) error {
	_, err := linkOf(c).Dismiss(ctx, []modulessdk.LinkDefinition{
		modulessdk.NewLinkDefinition(
			"blog", "post_id", postID,
			utilsmodulessdk.PRODUCT, "product_id", productID,
		),
	})
	return err
}

// endregion: dismiss

// region: list
// PostProductLinks lists the stored links of a post and a product.
func PostProductLinks(ctx context.Context, c Resolver, postID, productID string) ([]*dal.Record, error) {
	return linkOf(c).List(ctx, []modulessdk.LinkDefinition{
		modulessdk.NewLinkDefinition(
			"blog", "post_id", postID,
			utilsmodulessdk.PRODUCT, "product_id", productID,
		),
	}, false)
}

// endregion: list

// region: delete-cascade
// DeleteProductLinks soft-deletes the links of a product, and the records
// linked with a delete cascade.
func DeleteProductLinks(ctx context.Context, c Resolver, productID string) error {
	errs, _ := linkOf(c).Delete(ctx, modulessdk.DeleteEntityInput(utilsmodulessdk.PRODUCT, "product_id", productID))
	if len(errs) > 0 {
		return errs[0].Error
	}
	return nil
}

// endregion: delete-cascade

// region: restore
// RestoreProductLinks restores the links soft-deleted with a product.
func RestoreProductLinks(ctx context.Context, c Resolver, productID string) error {
	errs, _ := linkOf(c).Restore(ctx, modulessdk.DeleteEntityInput(utilsmodulessdk.PRODUCT, "product_id", productID))
	if len(errs) > 0 {
		return errs[0].Error
	}
	return nil
}

// endregion: restore

// region: workflow
// LinkPostInput is the input of LinkPostToProductWorkflow.
type LinkPostInput struct {
	PostID    string
	ProductID string
}

// LinkPostToProductWorkflow links a post to a product. If the workflow
// fails after the step that creates the link, the step's compensation
// dismisses the link again.
var LinkPostToProductWorkflow = composer.CreateWorkflow("link-post-to-product",
	func(e *composer.Execution, input LinkPostInput) ([]*dal.Record, error) {
		return commonsteps.CreateRemoteLinkStep.Run(e, []*dal.Record{
			modulessdk.NewLinkDefinition(
				"blog", "post_id", input.PostID,
				utilsmodulessdk.PRODUCT, "product_id", input.ProductID,
			),
		})
	})

// endregion: workflow

// region: link-definition
// postProductDefinition describes the link between a post and a product.
func postProductDefinition() modulessdk.LinkDefinition {
	return modulessdk.NewLinkDefinition(
		"blog", "post_id", "post_123",
		"product", "product_id", "prod_123",
	)
}

// endregion: link-definition
