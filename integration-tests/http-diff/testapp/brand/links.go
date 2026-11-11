package brand

import (
	"github.com/hitkey-io/medusago/packages/core/types/src/joiner"
	utilsmodulessdk "github.com/hitkey-io/medusago/packages/core/utils/src/modules-sdk"
	"github.com/hitkey-io/medusago/packages/medusa/src/modules"
)

func strPtr(s string) *string { return &s }

// DefineLinks mirrors src/links/brand.js: it registers the links of the
// brand module (call it once, like importing the links directory).
func DefineLinks() error {
	brand := Module.Linkable
	product := utilsmodulessdk.LinkableOf(&modules.Product)
	salesChannel := utilsmodulessdk.LinkableOf(&modules.SalesChannel)
	currency := utilsmodulessdk.LinkableOf(&modules.Currency)

	// product -> brand (a brand has many products)
	if _, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: product["product"].ToJSON(), IsList: true},
		brand["brand"].ToJSON(),
	); err != nil {
		return err
	}
	// brand <-> sales channels, with extra columns
	if _, err := utilsmodulessdk.DefineLink(
		brand["brand"].ToJSON(),
		utilsmodulessdk.DefineLinkInput{Linkable: salesChannel["salesChannel"].ToJSON(), IsList: true},
		utilsmodulessdk.DefineLinkOptions{Database: &utilsmodulessdk.LinkDatabaseOptions{
			Table: "brand_sales_channel",
			ExtraColumns: joiner.ExtraFields{
				{Name: "priority", ExtraField: joiner.ExtraField{Type: "integer", DefaultValue: strPtr("0")}},
				{Name: "featured", ExtraField: joiner.ExtraField{Type: "boolean", DefaultValue: strPtr("false")}},
				{Name: "note", ExtraField: joiner.ExtraField{Type: "text", Nullable: true}},
				{Name: "weight", ExtraField: joiner.ExtraField{Type: "decimal", Nullable: true}},
				{Name: "settings", ExtraField: joiner.ExtraField{Type: "json", Nullable: true}},
				{Name: "starts_at", ExtraField: joiner.ExtraField{Type: "datetime", Nullable: true}},
			},
		}},
	); err != nil {
		return err
	}
	// brand.currency_code -> currency (read-only)
	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: brand["brand"].ToJSON(), Field: "currency_code"},
		currency["currency"].ToJSON(),
		utilsmodulessdk.DefineLinkOptions{ReadOnly: true},
	)
	return err
}

// DefineIndexFilterableLink mirrors src/links/index-filterable.js: a link
// with filterable fields (MEDUSA_TEST_INDEX_FILTERABLE).
func DefineIndexFilterableLink() error {
	brand := Module.Linkable
	customer := utilsmodulessdk.LinkableOf(&modules.Customer)
	_, err := utilsmodulessdk.DefineLink(
		utilsmodulessdk.DefineLinkInput{Linkable: brand["brand"].ToJSON(), Filterable: []string{"name", "status", "rank", "launched_at"}},
		utilsmodulessdk.DefineLinkInput{Linkable: customer["customerGroup"].ToJSON(), IsList: true, Filterable: []string{"name"}},
	)
	return err
}
