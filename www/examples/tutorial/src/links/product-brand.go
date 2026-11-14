// Package links defines the module links of the project.
package links

import (
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/www/examples/tutorial/src/modules/brand"
)

// region: link
// Define defines the link between a product and a brand. A brand has many
// products, so the product side of the link is a list.
func Define() error {
	product := utilsmodulessdk.LinkableOf(&modules.Product)["product"]

	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: product.ToJSON(), IsList: true},
		brand.Module.Linkable["brand"].ToJSON(),
	)
	return err
}

// endregion: link
