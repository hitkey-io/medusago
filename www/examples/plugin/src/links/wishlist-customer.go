// Package links defines the module links of the plugin.
package links

import (
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/www/examples/plugin/src/modules/wishlist"
)

// region: body
// Define defines the links of the plugin: a wishlist belongs to a
// customer. The link is read-only, as the wishlist stores the customer's ID.
func Define() error {
	customer := utilsmodulessdk.LinkableOf(&modules.Customer)["customer"].ToJSON()

	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: wishlist.Module.Linkable["wishlist"].ToJSON(), Field: "customer_id"},
		customer,
		utilsmodulessdk.DefineLinkOptions{ReadOnly: true},
	)
	return err
}

// endregion: body
