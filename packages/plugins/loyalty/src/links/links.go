// Package links mirrors the links of @medusajs/loyalty-plugin (src/links),
// defined in the order the directory is read.
package links

import (
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
	"github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/loyalty"
	storecredit "github.com/hitkey-io/medusago/packages/plugins/loyalty/src/modules/store-credit"
)

// Define defines the links (like importing the files of src/links).
func Define() error {
	giftCard := loyalty.Module.Linkable["giftCard"].ToJSON()
	account := storecredit.Module.Linkable["storeCreditAccount"].ToJSON()
	cart := utilsmodulessdk.LinkableOf(&modules.Cart)["cart"].ToJSON()
	customer := utilsmodulessdk.LinkableOf(&modules.Customer)["customer"].ToJSON()
	order := utilsmodulessdk.LinkableOf(&modules.Order)["order"].ToJSON()

	// cart-gift-cards-link.ts
	if _, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: cart, IsList: true},
		utilsmodulessdk.DefineLinkInput{Linkable: giftCard, IsList: true},
	); err != nil {
		return err
	}
	// customer-store-credit-account-link.ts
	if _, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: account, Field: "customer_id"},
		customer,
		utilsmodulessdk.DefineLinkOptions{ReadOnly: true},
	); err != nil {
		return err
	}
	// gift-card-store-credit.ts
	if _, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: giftCard, IsList: true},
		account,
	); err != nil {
		return err
	}
	// order-gift-cards-link.ts
	if _, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: order, IsList: true},
		utilsmodulessdk.DefineLinkInput{Linkable: giftCard, IsList: true},
	); err != nil {
		return err
	}
	// order-line-item-gift-card-link.ts
	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: giftCard, Field: "line_item_id"},
		utilsmodulessdk.Linkable{ServiceName: "order", Entity: "OrderLineItem", Alias: "line_item", Field: "id", Linkable: "OrderLineItem", PrimaryKey: "id"},
		utilsmodulessdk.DefineLinkOptions{ReadOnly: true},
	)
	return err
}
